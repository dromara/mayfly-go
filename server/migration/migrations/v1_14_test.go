package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	dbentity "mayfly-go/internal/db/domain/entity"
)

// TestV1_14Idempotent 迁移体幂等：sqlite内存库上连续执行两次均成功，
// 且第二次执行不重复加列/建表
func TestV1_14Idempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 模拟存量库：t_db_transfer_task已存在但无concurrency列
	require.NoError(t, db.Table("t_db_transfer_task").AutoMigrate(&legacyDbTransferTask{}))

	migrations := V1_14()
	require.Len(t, migrations, 1)

	// 首次执行
	require.NoError(t, migrations[0].Migrate(db))
	// 重复执行（模拟重跑/多实例并发后再跑）
	require.NoError(t, migrations[0].Migrate(db))

	// concurrency列存在
	assert.True(t, db.Migrator().HasColumn(&dbentity.DbTransferTask{}, "concurrency"), "concurrency column should exist")
	// checkpoint表存在
	assert.True(t, db.Migrator().HasTable(&dbentity.DbTransferCheckpoint{}), "checkpoint table should exist")
}

// legacyDbTransferTask 模拟旧版本的迁移任务表（无concurrency列）
type legacyDbTransferTask struct {
	Id   uint64 `gorm:"primaryKey"`
	Name string
}
