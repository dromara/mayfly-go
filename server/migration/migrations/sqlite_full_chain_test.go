package migrations

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	aientity "mayfly-go/internal/ai/domain/entity"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/gormx"

	"github.com/glebarez/sqlite"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const itMysqlDSNEnv = "MAYFLY_IT_MIGRATION_MYSQL_DSN"

// openSqliteMigDB 打开迁移链端到端验证用的 SQLite 内存库。
// NamingStrategy 必须取生产的同一份，否则没写 TableName() 的实体会被解析到另一张表上
func openSqliteMigDB(t *testing.T) *gorm.DB {
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

// fullChainMigrations 从初装到最新版本的全部迁移，顺序与 migration.run 的注册顺序一致
func fullChainMigrations() []*gormigrate.Migration {
	return concatMigrations(Init, V1_9, V1_10, V1_11, V1_12)
}

// concatMigrations 按注册顺序拼接各版本的迁移切片
func concatMigrations(fs ...func() []*gormigrate.Migration) []*gormigrate.Migration {
	var ms []*gormigrate.Migration
	for _, f := range fs {
		ms = append(ms, f()...)
	}
	return ms
}

// openMysqlTempDB 在真实 MySQL 上开一个进程独占的随机临时库，返回库连接与「删库」收尾函数。
// adminDSN 不带库名（需要建库权限）
func openMysqlTempDB(t *testing.T, adminDSN string) (*gorm.DB, func()) {
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

func newChainMigrator(db *gorm.DB, ms []*gormigrate.Migration) *gormigrate.Gormigrate {
	return gormigrate.New(db, &gormigrate.Options{
		TableName:                 "migrations",
		IDColumnName:              "id",
		IDColumnSize:              300,
		UseTransaction:            true,
		ValidateUnknownMigrations: false,
	}, ms)
}

// 断言迁移链的关键落库事实：迁移全部记账（隐含迁移 ID 无重复）、
// 迁移直插的行审计字段真实落库、内置技能同步出了插件实例
func requireChainArtifacts(t *testing.T, db *gorm.DB, totalMigrations int) {
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

	for _, s := range builtinSkills {
		var instance aientity.PluginInstance
		require.NoError(t, db.Where("code = ?", s.Code).First(&instance).Error, "技能 %s 必须同步出插件实例", s.Code)
		require.Equal(t, aientity.PluginTypeSkill, instance.PluginType)
		require.Equal(t, s.Name, instance.Name)
		require.Equal(t, `{"skillCode":"`+s.Code+`"}`, instance.Config)
		require.NotNil(t, instance.CreateTime)
	}
}

// 完整迁移链（Init→V1_12）必须在全新 SQLite 上一次跑通。
// 迁移原生 SQL 只能用 MySQL 与 SQLite 的方言交集，任何单方言函数
// （NOW/CONCAT 等）都会让 SQLite 用户升级到一半 panic 回滚
func TestFullMigrationChainOnSqlite(t *testing.T) {
	db := openSqliteMigDB(t)

	ms := fullChainMigrations()
	m := newChainMigrator(db, ms)
	require.NoError(t, m.Migrate())

	requireChainArtifacts(t, db, len(ms))
}

// 迁移失败重试场景：失败迁移未记账，修复后函数体会再次执行，
// 因此每条迁移的函数体必须自身幂等（查重后跳过），直接重跑不能报错。
// Init/V1_9/V1_10/V1_11 的 seed 依赖记账防重跑，不在直调范围内
func TestV1_12MigrationsAreRerunnable(t *testing.T) {
	db := openSqliteMigDB(t)

	ms := fullChainMigrations()
	m := newChainMigrator(db, ms)
	require.NoError(t, m.Migrate())

	for _, migration := range V1_12() {
		require.NoError(t, migration.Migrate(db), "migration %s 重复执行必须幂等", migration.ID)
	}

	// 重跑后存量技能不得被重复同步成实例
	for _, s := range builtinSkills {
		var count int64
		require.NoError(t, db.Model(&aientity.PluginInstance{}).Where("code = ?", s.Code).Count(&count).Error)
		require.Equal(t, int64(1), count, "技能 %s 的插件实例必须只有一条", s.Code)
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
	adminDSN := os.Getenv(itMysqlDSNEnv)
	if adminDSN == "" {
		t.Skipf("set %s to run the full migration chain against a real mysql", itMysqlDSNEnv)
	}

	db, drop := openMysqlTempDB(t, adminDSN)
	defer drop()

	ms := fullChainMigrations()
	m := newChainMigrator(db, ms)
	require.NoError(t, m.Migrate())

	requireChainArtifacts(t, db, len(ms))

	for _, migration := range V1_12() {
		require.NoError(t, migration.Migrate(db), "migration %s 重复执行必须幂等", migration.ID)
	}

	// gormigrate 层面二次执行：已记账的迁移全部跳过
	require.NoError(t, m.Migrate())
	var migrationCount int64
	require.NoError(t, db.Table("migrations").Count(&migrationCount).Error)
	require.Equal(t, int64(len(ms)), migrationCount, "重复执行不得产生新的迁移记账")
}
