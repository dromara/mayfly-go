package migrations

import (
	"os"
	"testing"

	"mayfly-go/migration/migrations/migtest"
	v112 "mayfly-go/migration/migrations/v1_12"

	agentopsentity "mayfly-go/internal/agentops/domain/entity"

	"github.com/stretchr/testify/require"
)

// 完整迁移链（initdb → 最新版本）必须在全新 SQLite 上一次跑通。
// 迁移原生 SQL 只能用 MySQL 与 SQLite 的方言交集，任何单方言函数
// （NOW/CONCAT 等）都会让 SQLite 用户升级到一半 panic 回滚
func TestFullMigrationChainOnSqlite(t *testing.T) {
	db := migtest.OpenSqlite(t)

	ms := migtest.Concat(All()...)
	require.NoError(t, migtest.NewChainMigrator(db, ms).Migrate())

	migtest.RequireChainArtifacts(t, db, len(ms))
}

// 内置 Runbook 任务模板由 v1.12.2 批次 seed，链级落库事实单独断言：
// 模板数据是版本包内的私有定义，只有聚合层能同时拿到完整链与它
func TestBuiltinRunbooksSeededByChain(t *testing.T) {
	db := migtest.OpenSqlite(t)

	ms := migtest.Concat(All()...)
	require.NoError(t, migtest.NewChainMigrator(db, ms).Migrate())

	// 默认禁用落库、只读诊断档、admin 为默认负责人、Instruction 经 $skill-code 引用技能
	for _, rb := range v112.BuiltinRunbookTemplates {
		var task agentopsentity.AgentTask
		require.NoError(t, db.Where("key = ?", rb.Key).First(&task).Error, "Runbook 模板 %s 必须落库", rb.Key)
		require.Equal(t, agentopsentity.AgentTaskStatusDisable, task.Status, "Runbook 模板必须默认禁用")
		require.Equal(t, rb.ProfileId, task.ProfileId)
		require.Equal(t, uint64(1), task.OwnerAccountId, "默认负责人为 admin")
		require.Contains(t, task.Instruction, "$", "Instruction 须经 $skill-code 引用技能")
		require.NotNil(t, task.CreateTime)
		require.False(t, task.CreateTime.IsZero(), "create_time 必须由迁移写入真实时间")
	}
}

// TestRerunReseedsSoftDeletedResource 预置「用户删过内置菜单」的存量形态后重放迁移：
// 必须复活而不是撞主键 1062。
//
// 这条路径在全新库上跑不出来（没有软删行），只会在老库上把迁移链卡死；
// 而 t_sys_resource 的主键唯一性不含 is_deleted，所以「按未删除判存在」的守卫会走 INSERT 分支
func TestRerunReseedsSoftDeletedResource(t *testing.T) {
	db := migtest.OpenSqlite(t)

	ms := migtest.Concat(All()...)
	require.NoError(t, migtest.NewChainMigrator(db, ms).Migrate())

	// 用户在界面上删掉一个由 v1.12.2 内置的按钮权限（软删，主键仍占位）
	const code = "agentops:run:queryAll"
	require.NoError(t, db.Exec("UPDATE t_sys_resource SET is_deleted = 1 WHERE code = ?", code).Error)

	// 重放 v1.12 全版本条目（模拟迁移失败重试）：守卫若只看未删除行，这里会 INSERT 撞主键
	for _, migration := range v112.Migrations() {
		require.NoError(t, migration.Migrate(db), "migration %s 在存在软删资源行时必须幂等而非撞主键", migration.ID)
	}

	var stillDeleted int64
	require.NoError(t, db.Table("t_sys_resource").Where("code = ? AND is_deleted = 1", code).Count(&stillDeleted).Error)
	require.Zero(t, stillDeleted, "软删的内置资源必须被复活，否则升级后该按钮对所有人永久不可见")
}

// 迁移失败重试场景：失败迁移未记账，修复后函数体会再次执行，
// 因此每条迁移的函数体必须自身幂等（查重后跳过），直接重跑不能报错。
// initdb/v1_9/v1_10/v1_11 的 seed 依赖记账防重跑，不在直调范围内
func TestV1_12MigrationsAreRerunnable(t *testing.T) {
	db := migtest.OpenSqlite(t)

	ms := migtest.Concat(All()...)
	require.NoError(t, migtest.NewChainMigrator(db, ms).Migrate())

	// v112.Migrations() 已含同目录 v1_12_2.go 的 v1.12.2 批次，按发布顺序直调重跑
	for _, migration := range v112.Migrations() {
		require.NoError(t, migration.Migrate(db), "migration %s 重复执行必须幂等", migration.ID)
	}
}

// 完整迁移链必须在真实 MySQL 上跑通。MySQL 的 DDL 逐条隐式提交、事务无法整体回滚，
// 失败重试只能靠迁移函数体自身的幂等查重，这里一并直调重跑验证。
//
//	运行：MAYFLY_IT_MIGRATION_MYSQL_DSN='root:***@tcp(127.0.0.1:3306)/?charset=utf8&loc=Local&parseTime=true' \
//	       go test ./migration/migrations/ -run TestFullMigrationChainOnMysql
//
// DSN 不带库名；测试在进程独占的随机临时库里跑，结束即删库
func TestFullMigrationChainOnMysql(t *testing.T) {
	adminDSN := os.Getenv(migtest.ItMysqlDSNEnv)
	if adminDSN == "" {
		t.Skipf("set %s to run the full migration chain against a real mysql", migtest.ItMysqlDSNEnv)
	}

	db, drop := migtest.OpenMysqlTempDB(t, adminDSN)
	defer drop()

	ms := migtest.Concat(All()...)
	m := migtest.NewChainMigrator(db, ms)
	require.NoError(t, m.Migrate())

	migtest.RequireChainArtifacts(t, db, len(ms))

	for _, migration := range v112.Migrations() {
		require.NoError(t, migration.Migrate(db), "migration %s 重复执行必须幂等", migration.ID)
	}

	// gormigrate 层面二次执行：已记账的迁移全部跳过
	require.NoError(t, m.Migrate())
	var migrationCount int64
	require.NoError(t, db.Table("migrations").Count(&migrationCount).Error)
	require.Equal(t, int64(len(ms)), migrationCount, "重复执行不得产生新的迁移记账")
}
