package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"mayfly-go/internal/db/domain/entity"
)

func T20231125() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20231115",
		Migrate: func(tx *gorm.DB) error {
			entities := [...]any{
				new(entity.DbBackup),
				new(entity.DbBackupHistory),
				new(entity.DbRestore),
				new(entity.DbRestoreHistory),
				new(entity.DbBinlog),
				new(entity.DbBinlogHistory),
			}
			for _, e := range entities {
				if err := tx.AutoMigrate(e); err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
