package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	dbentity "mayfly-go/internal/db/domain/entity"
)

func V1_14() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			// DB迁移任务增强：并发度列（t_db_transfer_task.concurrency，阶段3实体已带gorm default:4 tag）
			// + 断点续传检查点表（t_db_transfer_checkpoint）。
			// 幂等：HasColumn/HasTable判重，已存在即跳过
			ID: "v1.14.0-db-transfer-concurrency-checkpoint",
			Migrate: func(tx *gorm.DB) error {
				if !tx.Migrator().HasColumn(&dbentity.DbTransferTask{}, "concurrency") {
					if err := tx.Migrator().AddColumn(&dbentity.DbTransferTask{}, "concurrency"); err != nil {
						return err
					}
				}
				if !tx.Migrator().HasTable(&dbentity.DbTransferCheckpoint{}) {
					if err := tx.AutoMigrate(&dbentity.DbTransferCheckpoint{}); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}
