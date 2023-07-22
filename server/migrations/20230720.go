package migrations

import (
	authentity "mayfly-go/internal/auth/domain/entity"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// T20230720 三方登录表
func T20230720() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20230720",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&authentity.Oauth2Account{})
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
