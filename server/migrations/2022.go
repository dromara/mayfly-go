package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// T2022 TODO 在此之前的数据库表结构初始化, 目前先使用mayfly-go.sql文件初始化数据库结构
func T2022() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "2022",
		Migrate: func(tx *gorm.DB) error {
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
