// Package migtest 迁移链的测试基建：开库、跑链、断言链级落库事实。
//
// 单独成包的原因：版本目录化后，各版本包自己的测试也要开库跑迁移
// （例如 v1.12.2 的「先回填再删列」用例，步骤体会遍历四张 agentops 表，
// 缺表时 Migrator 直接报错而不是跳过），而 Go 的 _test.go 文件不能被其他包 import。
//
// 本包只依赖实体与基座，不 import migrations：否则版本包的测试会因
// 「migrations → 版本包 → 测试依赖 migtest → migrations」构成导入环。链内容由调用方喂 Concat
//
// 本包只被测试代码引用，不参与生产链接。
package migtest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	agentopsentity "mayfly-go/internal/agentops/domain/entity"
	aientity "mayfly-go/internal/ai/domain/entity"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/migration/migrations/migrationx"
	"mayfly-go/pkg/gormx"

	"github.com/glebarez/sqlite"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ItMysqlDSNEnv 真实 MySQL 集成测试的 admin DSN 环境变量（不带库名，需要建库权限）
const ItMysqlDSNEnv = "MAYFLY_IT_MIGRATION_MYSQL_DSN"

// OpenSqlite 打开迁移链端到端验证用的 SQLite 内存库。
// NamingStrategy 必须取生产的同一份，否则没写 TableName() 的实体会被解析到另一张表上
func OpenSqlite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableAutomaticPing: true,
		NamingStrategy:       gormx.MetaNamingStrategy(),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// :memory: 每个连接是独立库，锁死单连接避免建表与写入落在不同连接
	sqlDB.SetMaxOpenConns(1)
	return db
}

// Concat 按注册顺序拼接各版本的迁移切片。
// 链的内容由调用方传入（通常是 migrations.All()），本包不依赖顺序真源
func Concat(batches ...func() []*gormigrate.Migration) []*gormigrate.Migration {
	var ms []*gormigrate.Migration
	for _, batch := range batches {
		ms = append(ms, batch()...)
	}
	return ms
}

// NewChainMigrator 复刻 gormigrate.Options 与生产一致（含 UseTransaction 与不校验未知迁移）
func NewChainMigrator(db *gorm.DB, ms []*gormigrate.Migration) *gormigrate.Gormigrate {
	return gormigrate.New(db, &gormigrate.Options{
		TableName:                 "migrations",
		IDColumnName:              "id",
		IDColumnSize:              300,
		UseTransaction:            true,
		ValidateUnknownMigrations: false,
	}, ms)
}

// OpenMysqlTempDB 在真实 MySQL 上开一个进程独占的随机临时库，返回库连接与「删库」收尾函数
func OpenMysqlTempDB(t *testing.T, adminDSN string) (*gorm.DB, func()) {
	t.Helper()
	admin, err := gorm.Open(mysql.Open(adminDSN), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		NamingStrategy: gormx.MetaNamingStrategy(),
	})
	require.NoError(t, err)

	// 库名由本测试生成的纳秒时间戳常量拼接，无外部输入面
	dbName := fmt.Sprintf("mayfly_mig_it_%d", time.Now().UnixNano())
	require.NoError(t, admin.Exec("CREATE DATABASE `"+dbName+"` CHARACTER SET utf8mb4").Error)

	dbDSN := strings.Replace(adminDSN, "/?", "/"+dbName+"?", 1)
	db, err := gorm.Open(mysql.Open(dbDSN), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		NamingStrategy: gormx.MetaNamingStrategy(),
	})
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, admin.Exec("DROP DATABASE IF EXISTS `"+dbName+"`").Error)
	}
}

// RequireChainArtifacts 断言迁移链的链级落库事实：迁移全部记账（隐含迁移 ID 无重复）、
// 迁移直插的行审计字段真实落库、内置技能同步出了插件实例、AgentOps 表齐且旧展示列已收敛进 extra。
//
// 批次专属数据（如内置 Runbook 模板）的断言不在这里——那需要引用版本包内的私有数据，
// 由各版本包自己的测试用例负责
func RequireChainArtifacts(t *testing.T, db *gorm.DB, totalMigrations int) {
	t.Helper()

	var migrationCount int64
	require.NoError(t, db.Table("migrations").Count(&migrationCount).Error)
	require.Equal(t, int64(totalMigrations), migrationCount,
		"每个迁移都必须留下记账记录，少一条说明存在重复 ID 或未执行项")

	var granted sysentity.RoleResource
	require.NoError(t, db.Where("role_id = 1 AND resource_id = 1775967861").First(&granted).Error)
	require.NotNil(t, granted.CreateTime)
	require.False(t, granted.CreateTime.IsZero(), "create_time 必须由迁移写入真实时间")
	require.Equal(t, "admin", granted.Creator)

	for _, s := range migrationx.BuiltinSkills {
		var instance aientity.PluginInstance
		require.NoError(t, db.Where("code = ?", s.Code).First(&instance).Error, "技能 %s 必须同步出插件实例", s.Code)
		require.Equal(t, aientity.PluginTypeSkill, instance.PluginType)
		require.Equal(t, s.Name, instance.Name)
		require.Equal(t, `{"skillCode":"`+s.Code+`"}`, instance.Config)
		require.NotNil(t, instance.CreateTime)
	}

	// 事件驱动触发器表：阶段2 触发源（alert/webhook）落库依赖 t_aiops_trigger 存在
	require.True(t, db.Migrator().HasTable(&agentopsentity.Trigger{}),
		"迁移链必须建出触发器表 t_aiops_trigger")

	// 变更提案表：阶段3 自主运维闭环（提案→审批→受控执行→验证）落库依赖 t_aiops_proposal 存在
	require.True(t, db.Migrator().HasTable(&agentopsentity.Proposal{}),
		"迁移链必须建出变更提案表 t_aiops_proposal")

	// 展示型字段已收敛进 extra（v1.12.2）：整条链跑完后，旧展示列不得存在于任何一张 agentops 表，
	// 而 extra 列必须在（它是这些值的唯一落库处）。否则新代码写入的展示值无处可存，而旧列又以 NULL 残留
	for _, legacy := range []struct {
		name    string
		dest    any
		columns []string
	}{
		{"t_aiops_run", &agentopsentity.Run{}, []string{"owner_username", "feedback_note"}},
		{"t_aiops_task", &agentopsentity.AgentTask{}, []string{"owner_username", "remark", "target_resources"}},
		{"t_aiops_trigger", &agentopsentity.Trigger{}, []string{"owner_username", "remark"}},
		{"t_aiops_proposal", &agentopsentity.Proposal{}, []string{"owner_username"}},
	} {
		for _, column := range legacy.columns {
			require.False(t, db.Migrator().HasColumn(legacy.dest, column),
				"%s.%s 应已被 v1.12.2 删列（展示字段收敛进 extra）", legacy.name, column)
		}
		require.True(t, db.Migrator().HasColumn(legacy.dest, "extra"),
			"%s 必须有 extra 列承载展示字段", legacy.name)
	}
}
