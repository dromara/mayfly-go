package migrations

import (
	"testing"

	dbentity "mayfly-go/internal/db/domain/entity"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// cronRenameMigration 取出定时列改名迁移，供测试直接执行
func cronRenameMigration(t *testing.T) func(*gorm.DB) error {
	t.Helper()
	for _, m := range V1_12() {
		if m.ID == "v1.12.0-db-transfer-cron-column-rename" {
			return m.Migrate
		}
	}
	t.Fatal("cron column rename migration not found")
	return nil
}

func openCronMigDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	return db
}

// createLegacyTransferTaskTable 按旧列名 cron_able 建表，并插入一条已开启定时的任务
func createLegacyTransferTaskTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("CREATE TABLE t_db_transfer_task (id INTEGER PRIMARY KEY, task_name TEXT, cron_able INTEGER NOT NULL DEFAULT -1)").Error; err != nil {
		t.Fatalf("create legacy table failed: %v", err)
	}
	if err := db.Exec("INSERT INTO t_db_transfer_task (id, task_name, cron_able) VALUES (1, 't1', 1)").Error; err != nil {
		t.Fatalf("insert legacy row failed: %v", err)
	}
}

func cronEnabledValue(t *testing.T, db *gorm.DB) int8 {
	t.Helper()
	var value int8
	if err := db.Table("t_db_transfer_task").Where("id = 1").Select("cron_enabled").Scan(&value).Error; err != nil {
		t.Fatalf("read cron_enabled failed: %v", err)
	}
	return value
}

func TestMigrateDbTransferCronColumnRename(t *testing.T) {
	migrate := cronRenameMigration(t)

	t.Run("旧库改名并保留数据", func(t *testing.T) {
		db := openCronMigDB(t)
		createLegacyTransferTaskTable(t, db)

		if err := migrate(db); err != nil {
			t.Fatalf("rename cron_able to cron_enabled failed: %v", err)
		}
		if db.Migrator().HasColumn(&dbentity.DbTransferTask{}, "cron_able") {
			t.Fatal("old column cron_able should be dropped")
		}
		if got := cronEnabledValue(t, db); got != 1 {
			t.Fatalf("cron_enabled should keep legacy value 1, got %d", got)
		}

		// 迁移后列名与实体字段名一致，查询条件体可直接传查询dto（按字段名推导列名）
		var tasks []*dbentity.DbTransferTask
		if err := db.Model(&dbentity.DbTransferTask{}).Where(&dbentity.DbTransferTaskQuery{CronEnabled: dbentity.DbTransferTaskCronEnabled}).
			Select("id").Find(&tasks).Error; err != nil {
			t.Fatalf("query with cron_enabled column failed: %v", err)
		}
		if len(tasks) != 1 {
			t.Fatalf("enabled cron task count should be 1, got %d", len(tasks))
		}
	})

	t.Run("重复执行幂等", func(t *testing.T) {
		db := openCronMigDB(t)
		createLegacyTransferTaskTable(t, db)

		if err := migrate(db); err != nil {
			t.Fatalf("first migrate failed: %v", err)
		}
		if err := migrate(db); err != nil {
			t.Fatalf("second migrate should be no-op, got: %v", err)
		}
		if got := cronEnabledValue(t, db); got != 1 {
			t.Fatalf("cron_enabled should stay 1, got %d", got)
		}
	})

	t.Run("全新库自动建列无需处理", func(t *testing.T) {
		db := openCronMigDB(t)
		if err := db.AutoMigrate(new(dbentity.DbTransferTask)); err != nil {
			t.Fatalf("automigrate failed: %v", err)
		}
		if db.Migrator().HasColumn(&dbentity.DbTransferTask{}, "cron_able") {
			t.Fatal("fresh table should not contain legacy column cron_able")
		}
		if err := migrate(db); err != nil {
			t.Fatalf("migrate on fresh table failed: %v", err)
		}
	})

	t.Run("两列并存时搬运数据后删旧列", func(t *testing.T) {
		db := openCronMigDB(t)
		createLegacyTransferTaskTable(t, db)
		// 模拟历史版本以别名映射过：新列已被自动补建（默认 -1 即未开启），旧列仍有真实值
		if err := db.Exec("ALTER TABLE t_db_transfer_task ADD COLUMN cron_enabled INTEGER NOT NULL DEFAULT -1").Error; err != nil {
			t.Fatalf("add cron_enabled failed: %v", err)
		}
		if got := cronEnabledValue(t, db); got != -1 {
			t.Fatalf("new column should hold default -1 before migrate, got %d", got)
		}

		if err := migrate(db); err != nil {
			t.Fatalf("migrate with both columns failed: %v", err)
		}
		if db.Migrator().HasColumn(&dbentity.DbTransferTask{}, "cron_able") {
			t.Fatal("legacy column cron_able should be dropped")
		}
		if got := cronEnabledValue(t, db); got != 1 {
			t.Fatalf("cron_enabled should be backfilled from cron_able, got %d", got)
		}
	})
}
