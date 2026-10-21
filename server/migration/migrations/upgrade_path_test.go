package migrations

import (
	"os"
	"strings"
	"testing"
	"time"

	dbentity "mayfly-go/internal/db/domain/entity"
	fileentity "mayfly-go/internal/file/domain/entity"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/migration/migrations/initdb"
	"mayfly-go/migration/migrations/migtest"
	"mayfly-go/migration/migrations/v1_10"
	"mayfly-go/migration/migrations/v1_11"
	"mayfly-go/migration/migrations/v1_12"
	"mayfly-go/migration/migrations/v1_9"
	"mayfly-go/pkg/model"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 本文件验证「升级」路径，与 sqlite_full_chain_test.go 的「全新装」路径互补。
//
// 全新库上 Init 直接按当前实体建表，因此「把已有列改成另一种类型」这条分支永远走不到——
// 而这恰恰是 SQLite 用户升级时炸掉的地方：SQLite 没有 ALTER COLUMN，驱动只能建新表、
// 搬数据、删旧表、改名，任何一步失败都会让迁移报错回滚，服务启动即退出。
// 所以这里先把旧版本的表结构原样建出来（含数据与索引），再跑本次新增的迁移。

// v1121MigrationIds 本次交付新增的迁移 ID。统一 v1.12.1- 前缀是发版约定：
// v1.12.0 批次已随上一版发布，跑过它的库会按 ID 跳过，新动作必须用新 ID 才会真的执行
var v1121MigrationIds = []string{
	"v1.12.1-file-access-mode",
	"v1.12.1-db-session-resources",
	"v1.12.1-db-schema-baseline",
	"v1.12.1-flow-procinst-planned-exec-time",
	"v1.12.1-db-sql-exec-row-snapshot",
}

// legacyDbSQLExec v1.12.1 之前 t_db_sql_exec 的实体形态（字段与 gorm tag 逐字对应旧版定义：
// old_value 是 size:5000、没有 new_value）。
//
// 必须用实体 + AutoMigrate 建旧表，不能手写 DDL：SQLite 用户的旧库本来就是旧版代码
// AutoMigrate 生成的，而驱动改列类型时会先 parse 原表 DDL 再拼出新表（建新表→搬交集列→
// 删旧表→改名）；手写 DDL 的排版差异会让它 parse 出另一种列集合，实测会隔一列丢一列，
// 搬数据时漏掉 db_id 直接撞 NOT NULL 约束——那测的就不是用户的升级路径了
type legacyDbSQLExec struct {
	model.Model

	DbId       uint64 `json:"dbId" gorm:"not null;"`
	Db         string `json:"db" gorm:"size:150;not null;"`
	Table      string `json:"table" gorm:"size:150;"`
	Type       int8   `json:"type" gorm:"not null;"`
	SQL        string `json:"sql" gorm:"column:sql;size:5000;not null;"`
	OldValue   string `json:"oldValue" gorm:"size:5000;"`
	Remark     string `json:"remark" gorm:"size:255;"`
	Status     int8   `json:"status"`
	Res        string `json:"res" gorm:"size:1000;"`
	FlowBizKey string `json:"flowBizKey" gorm:"size:50;index:idx_flow_biz_key;comment:流程关联的业务key"`
}

func (legacyDbSQLExec) TableName() string { return "t_db_sql_exec" }

// seedLegacySqlExec 往旧结构里塞一行存量旧值：升级后必须原样读回来，
// 否则说明驱动搬数据时把内容截断或丢了。
//
// table 与 sql 两列必须加反引号：它们在 MySQL 8.0 上是保留字，裸写直接报 1064
// （真库实测）。GORM 生成 DDL 时会自己加，但手写 raw SQL 得自己负责
func seedLegacySqlExec(t *testing.T, db *gorm.DB, oldValue string) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Exec(
		"INSERT INTO t_db_sql_exec (db_id, db, `table`, type, `sql`, old_value, status, create_time, creator, creator_id, update_time, modifier, modifier_id, is_deleted, flow_biz_key) "+
			"VALUES (1, 'testdb', 't_demo', 1, 'update t_demo set a=1', ?, 2, ?, 'admin', 1, ?, 'admin', 1, 0, 'biz-key-1')",
		oldValue, now, now,
	).Error)
}

// columnTypes 取表的「列名 → 声明类型」。表名来自本文件常量，无外部输入面
func columnTypes(t *testing.T, db *gorm.DB, table string) map[string]string {
	t.Helper()
	res := map[string]string{}
	if db.Dialector.Name() == "sqlite" {
		rows, err := db.Raw("PRAGMA table_info(" + table + ")").Rows()
		require.NoError(t, err)
		defer rows.Close()
		for rows.Next() {
			var cid, notnull, pk int
			var name, ctype string
			var dflt *string
			require.NoError(t, rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk))
			res[name] = ctype
		}
		return res
	}
	type col struct {
		Name string
		Type string
	}
	var cols []col
	require.NoError(t, db.Raw(
		"SELECT column_name AS name, column_type AS type FROM information_schema.columns "+
			"WHERE table_schema = DATABASE() AND table_name = ?", table).Scan(&cols).Error)
	for _, c := range cols {
		res[c.Name] = c.Type
	}
	return res
}

// runV1121Migrations 只跑本次新增的迁移（按 ID 取，不依赖切片下标）
func runV1121Migrations(t *testing.T, db *gorm.DB) {
	t.Helper()
	want := map[string]bool{}
	for _, id := range v1121MigrationIds {
		want[id] = true
	}
	found := 0
	for _, m := range v112.Migrations() {
		if !want[m.ID] {
			continue
		}
		found++
		require.NoError(t, m.Migrate(db), "迁移 %s 在旧结构库上必须能执行", m.ID)
	}
	require.Equal(t, len(want), found, "本次新增的迁移 ID 必须都在 V1_12 里，且统一以 v1.12.1- 开头")
}

// requireSqlExecUpgraded 断言 t_db_sql_exec 升级到位：新列存在、旧值列已是大文本、数据与索引未丢
func requireSqlExecUpgraded(t *testing.T, db *gorm.DB, oldValue string) {
	t.Helper()
	cols := columnTypes(t, db, "t_db_sql_exec")

	_, hasNew := cols["new_value"]
	require.True(t, hasNew, "升级后必须新增 new_value 列，实际列：%v", cols)

	// 列类型断言只在 MySQL 上成立，不要写成两边通用的「不含 varchar(5000)」：
	// GORM 的 sqlite dialector 对 size:5000 的 string 一律生成 TEXT（实测），所以 SQLite 旧库
	// 本来就没有长度上限，mediumtext 与 text 在它那里是同一种亲和性、无可观测差异
	//（把实体 tag 改回 size:5000 后 SQLite 用例仍通过，那是等价变异而不是断言漏写）。
	// SQLite 上真正会炸的是「改列类型 = 重建整张表」这件事本身，由下面的数据与索引断言守住
	if db.Dialector.Name() == "mysql" {
		require.NotContains(t, strings.ToLower(cols["old_value"]), "varchar",
			"old_value 必须已从 varchar 扩为大文本，实际类型：%s", cols["old_value"])
		require.NotContains(t, strings.ToLower(cols["new_value"]), "varchar",
			"new_value 必须是大文本类型，实际类型：%s", cols["new_value"])
	}

	// 数据必须原样搬过来：驱动重建表时漏搬或截断都会在这里现形
	var got dbentity.DbSQLExec
	require.NoError(t, db.Where("flow_biz_key = ?", "biz-key-1").First(&got).Error)
	require.Equal(t, oldValue, got.OldValue, "升级不得丢失或截断 old_value 内容")
	require.Empty(t, got.NewValue)

	// 索引必须跟着重建：SQLite 的 recreate 流程若不搬索引，升级后按业务 key 查执行记录会退化成全表扫
	require.True(t, db.Migrator().HasIndex(&dbentity.DbSQLExec{}, "FlowBizKey"),
		"升级后 idx_flow_biz_key 索引必须仍然存在")
}

// requireOversizeSnapshotFits 扩容必须真的生效：旧上限存不下的行级快照，升级后要能完整写入并原样读回。
// 只看列类型不够——改列类型静默失败时列名照样在，而写入会在超限时才炸
// （那时丢的是一整条执行记录，连审计一起）。写新行而不改存量行，重复执行本断言才能幂等
func requireOversizeSnapshotFits(t *testing.T, db *gorm.DB) {
	t.Helper()
	oversize := strings.Repeat("快照-", 30000) // 9 万字符，远超 varchar(5000) 的上限
	row := &dbentity.DbSQLExec{
		DbId: 1, Db: "testdb", Table: "t_demo", Type: dbentity.DbSQLExecTypeUpdate,
		SQL: "update t_demo set a=1", NewValue: oversize, FlowBizKey: "oversize-probe",
		Status: dbentity.DbSQLExecStatusSuccess,
	}
	// 审计列（create_time 等）在本项目是 not null 且由 FillBaseInfo 填，不靠 GORM 的 autoCreateTime；
	// 这里没有登录上下文，按后台任务的口径回落到系统账号
	row.FillBaseInfo(model.SysAccount)
	// 先清后写：本断言会在幂等复验时被调第二次，不得因探针行重复而拿到旧行
	require.NoError(t, db.Where("flow_biz_key = ?", "oversize-probe").Delete(&dbentity.DbSQLExec{}).Error)
	require.NoError(t, db.Create(row).Error)

	var after dbentity.DbSQLExec
	require.NoError(t, db.Where("flow_biz_key = ?", "oversize-probe").First(&after).Error)
	require.Equal(t, oversize, after.NewValue, "扩容后必须存得下超长的行级快照")
}

// requireNoUnknownAccessMode 存量文件行必须全部归为 public 或 private。
// 若留在零值，运行期按「未知即私有」处理，等于升级把已有头像全打挂
func requireNoUnknownAccessMode(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.Zero(t, unknownAccessModeCount(t, db), "存量文件行的 access_mode 不得留未知取值")
}

// unknownAccessModeCount 统计 access_mode 落在合法取值之外的行数
func unknownAccessModeCount(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var count int
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM t_sys_file WHERE access_mode IS NULL OR access_mode NOT IN (?, ?)",
		fileentity.FileAccessModePublic, fileentity.FileAccessModePrivate).Row().Scan(&count))
	return count
}

// upgradeFixture 造一个「停在 v1.12.0 的老库」。
//
// 不能只是跑 Init→V1_11 就完事：Init 是按**当前**实体 AutoMigrate 建表的，跑完之后
// 表结构已经是新版，改列类型那条分支根本不会触发（全新库测试正是因此漏掉了升级风险）。
// 所以这里先跑完旧链，再把本次要改的三张表**降级**回 v1.12.0 时代的形态：
//   - t_db_sql_exec：old_value 是 varchar(5000)、无 new_value
//   - t_sys_file：无 access_mode
//   - t_flow_procinst：无 planned_exec_time
func upgradeFixture(t *testing.T, db *gorm.DB, longValue string) {
	t.Helper()
	require.NoError(t, migtest.NewChainMigrator(db, migtest.Concat(initdb.Migrations, v19.Migrations, v110.Migrations, v111.Migrations, v112.Migrations)).Migrate())

	// 降级执行记录表：改列类型无法用 ALTER 表达（sqlite 根本没有 ALTER COLUMN），只能按旧实体重建
	require.NoError(t, db.Exec("DROP TABLE IF EXISTS t_db_sql_exec").Error)
	require.NoError(t, db.AutoMigrate(&legacyDbSQLExec{}))
	seedLegacySqlExec(t, db, longValue)

	// 降级另两张表：删列语法 sqlite 3.35+ 与 mysql 均原生支持
	for _, stmt := range []string{
		"ALTER TABLE t_sys_file DROP COLUMN access_mode",
		"ALTER TABLE t_flow_procinst DROP COLUMN planned_exec_time",
	} {
		require.NoError(t, db.Exec(stmt).Error, "降级语句执行失败：%s", stmt)
	}

	// 前置条件：老库里确实还没有本次要加的列，否则测的就不是升级路径
	require.False(t, db.Migrator().HasColumn(&dbentity.DbSQLExec{}, "new_value"),
		"降级后不该有 new_value 列，测试前提已失效")
	require.False(t, db.Migrator().HasColumn(&fileentity.File{}, "access_mode"),
		"降级后不该有 access_mode 列，测试前提已失效")
	require.False(t, db.Migrator().HasColumn(&flowentity.Procinst{}, "planned_exec_time"),
		"降级后不该有 planned_exec_time 列，测试前提已失效")

	// 记账表里抹掉本次要跑的迁移，让 gormigrate 也把它们当「未执行」——
	// 这正是真实升级时的状态：用户库里只记着老版本的 ID
	for _, id := range v1121MigrationIds {
		require.NoError(t, db.Exec("DELETE FROM migrations WHERE id = ?", id).Error)
	}
}

// legacyOldValue 旧库里的存量旧值：必须落在 varchar(5000) 以内（MySQL 会强制截断超长值，
// 而 SQLite 不校验长度，两边得用同一个合法输入）。
// 「扩容后存得下超大值」由 requireSqlExecUpgraded 在升级完成后单独验证
func legacyOldValue() string {
	return strings.Repeat("旧值-", 1300)
}

// TestUpgradeFromLegacySchemaOnSqlite 旧结构 SQLite 库升级到 v1.12.1 必须一次跑通且不丢数据。
//
// 这是 SQLite 用户「升级后启动失败」的复现路径：改列类型在 SQLite 上等于重建整张表
func TestUpgradeFromLegacySchemaOnSqlite(t *testing.T) {
	db := migtest.OpenSqlite(t)
	longValue := legacyOldValue()
	upgradeFixture(t, db, longValue)

	// 走 gormigrate 正式入口：既验证迁移体能执行，也验证记账与二次启动跳过
	require.NoError(t, migtest.NewChainMigrator(db, migtest.Concat(All()...)).Migrate())
	requireSqlExecUpgraded(t, db, longValue)
	requireOversizeSnapshotFits(t, db)
	requireNoUnknownAccessMode(t, db)

	// 迁移失败重试会再跑一次函数体（未记账的项），因此重复执行必须幂等；
	// 这里也顺带覆盖「已升级库再启动一次」的正常场景
	runV1121Migrations(t, db)
	requireSqlExecUpgraded(t, db, longValue)
	requireOversizeSnapshotFits(t, db)
	requireNoUnknownAccessMode(t, db)
}

// TestUpgradeFromLegacySchemaOnMysql 同一升级路径在真实 MySQL 上复验。
// MySQL 的 DDL 逐条隐式提交、事务无法整体回滚，失败重试只能靠迁移函数体自身查重
//
//	运行：MAYFLY_IT_MIGRATION_MYSQL_DSN='root:***@tcp(127.0.0.1:3306)/?charset=utf8&loc=Local&parseTime=true' \
//	       go test ./migration/migrations/ -run TestUpgradeFromLegacySchemaOnMysql
func TestUpgradeFromLegacySchemaOnMysql(t *testing.T) {
	adminDSN := os.Getenv(migtest.ItMysqlDSNEnv)
	if adminDSN == "" {
		t.Skipf("set %s to run the upgrade path against a real mysql", migtest.ItMysqlDSNEnv)
	}
	db, drop := migtest.OpenMysqlTempDB(t, adminDSN)
	defer drop()

	longValue := legacyOldValue()
	upgradeFixture(t, db, longValue)

	require.NoError(t, migtest.NewChainMigrator(db, migtest.Concat(All()...)).Migrate())
	requireSqlExecUpgraded(t, db, longValue)
	requireOversizeSnapshotFits(t, db)
	requireNoUnknownAccessMode(t, db)

	runV1121Migrations(t, db)
	requireSqlExecUpgraded(t, db, longValue)
	requireOversizeSnapshotFits(t, db)
	requireNoUnknownAccessMode(t, db)
}

// TestV1121MigrationIdsAreUniqueAndPrefixed 发版约定的机器可验证形式：
// 本次新增迁移一律 v1.12.1- 前缀，且整条链上不存在重复 ID（重复 ID 会让记账数对不上、
// 后一条永远不执行，是「代码改了但库没变」的典型成因）
func TestV1121MigrationIdsAreUniqueAndPrefixed(t *testing.T) {
	for _, id := range v1121MigrationIds {
		require.True(t, strings.HasPrefix(id, "v1.12.1-"), "本次新增迁移 ID 必须以 v1.12.1- 开头，实际：%s", id)
	}

	seen := map[string]int{}
	for _, m := range migtest.Concat(All()...) {
		seen[m.ID]++
	}
	for id, n := range seen {
		require.Equal(t, 1, n, "迁移 ID %s 重复了 %d 次", id, n)
	}
	for _, id := range v1121MigrationIds {
		require.Contains(t, seen, id, "迁移 %s 必须已注册进迁移链", id)
	}
}

// TestPlannedExecTimeColumnIsAddedOnce planned_exec_time 只由这一条迁移负责，
// 且重复执行不得报「列已存在」
func TestPlannedExecTimeColumnIsAddedOnce(t *testing.T) {
	db := migtest.OpenSqlite(t)
	upgradeFixture(t, db, "short")

	var target *gormigrate.Migration
	for _, m := range v112.Migrations() {
		if m.ID == "v1.12.1-flow-procinst-planned-exec-time" {
			target = m
		}
	}
	require.NotNil(t, target, "未找到 planned_exec_time 迁移")

	for i := 0; i < 3; i++ {
		require.NoError(t, target.Migrate(db), "第 %d 次执行必须成功", i+1)
	}
	require.True(t, db.Migrator().HasColumn(&flowentity.Procinst{}, "planned_exec_time"))
}

// TestFileAccessModeNormalizesUnknownValues 存量行若落在非法取值上（跑过中间构建、
// 或某方言加列时没按 default 回填），迁移里的归一 UPDATE 必须把它拉回 public。
//
// 正常升级路径上这条 UPDATE 是**冗余**的：列本来不存在，AutoMigrate 加的是
// not null default 1，存量行当场就是 1。所以必须单独造出非法取值才测得到它——
// 否则断言永真，把迁移里的 UPDATE 整条删掉也不会有人发现
func TestFileAccessModeNormalizesUnknownValues(t *testing.T) {
	db := migtest.OpenSqlite(t)
	upgradeFixture(t, db, "short")
	require.NoError(t, migtest.NewChainMigrator(db, migtest.Concat(All()...)).Migrate())

	probe := &fileentity.File{
		FileKey: "k-probe", Filename: "avatar.png", Path: "/tmp/avatar.png",
		Size: 1, StorageType: "local",
	}
	probe.FillBaseInfo(model.SysAccount)
	require.NoError(t, db.Create(probe).Error)

	// 造出「列已在但取值非法」的库形态：0 既非 public 也非 private，
	// 运行期按 fail-closed 当私有处理，头像会当场读不到
	require.NoError(t, db.Exec("UPDATE t_sys_file SET access_mode = 0").Error)
	require.Positive(t, unknownAccessModeCount(t, db),
		"用例前提：必须先造出非法取值的行，否则测的是空集")

	for _, m := range v112.Migrations() {
		if m.ID == "v1.12.1-file-access-mode" {
			require.NoError(t, m.Migrate(db))
		}
	}
	requireNoUnknownAccessMode(t, db)

	var after fileentity.File
	require.NoError(t, db.Where("file_key = ?", "k-probe").First(&after).Error)
	require.True(t, after.IsPublic(), "归一后存量文件必须能匿名读取，否则升级把已有头像打挂")
}
