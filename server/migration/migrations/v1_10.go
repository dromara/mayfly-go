package migrations

import (
	flowentity "mayfly-go/internal/flow/domain/entity"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func V1_10() []*gormigrate.Migration {
	var migrations []*gormigrate.Migration
	migrations = append(migrations, V1_10_0()...)
	return migrations
}

func V1_10_0() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250213-v1.10.0-flow-recode",
			Migrate: func(tx *gorm.DB) error {
				err := tx.AutoMigrate(&flowentity.Procdef{},
					&flowentity.Procinst{},
					&flowentity.Execution{},
					&flowentity.ProcinstTask{},
					flowentity.ProcinstTaskCandidate{},
					&flowentity.HisProcinstOp{})
				if err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}
