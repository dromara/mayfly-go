package migrations

import (
	entity2 "mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/machine/domain/entity"
	entity3 "mayfly-go/internal/mongo/domain/entity"
	entity6 "mayfly-go/internal/msg/domain/entity"
	entity4 "mayfly-go/internal/redis/domain/entity"
	entity5 "mayfly-go/internal/sys/domain/entity"
	entity7 "mayfly-go/internal/tag/domain/entity"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// T2022 TODO 在此之前的数据库表结构初始化, 目前先使用mayfly-go.sql文件初始化数据库结构
func T2022() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "2022",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&entity.AuthCert{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity.Machine{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity.MachineFile{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity.MachineMonitor{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity.MachineScript{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity.MachineCronJob{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity.MachineCronJobExec{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity.MachineCronJobRelate{}); err != nil {
				return err
			}

			if err := tx.AutoMigrate(&entity2.DbInstance{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity2.Db{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity2.DbSql{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity2.DbSqlExec{}); err != nil {
				return err
			}

			if err := tx.AutoMigrate(&entity3.Mongo{}); err != nil {
				return err
			}

			if err := tx.AutoMigrate(&entity4.Redis{}); err != nil {
				return err
			}

			if err := tx.AutoMigrate(&entity5.Account{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity5.AccountRole{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity5.Config{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity5.SysLog{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity5.Resource{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity5.Role{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity5.RoleResource{}); err != nil {
				return err
			}

			if err := tx.AutoMigrate(&entity6.Msg{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity7.TagTree{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity7.TagTreeTeam{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity7.Team{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&entity7.TeamMember{}); err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
