package migrations

import (
	machineentity "mayfly-go/internal/machine/domain/entity"
	sysentity "mayfly-go/internal/sys/domain/entity"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func V1_9() []*gormigrate.Migration {
	var migrations []*gormigrate.Migration
	migrations = append(migrations, V1_9_3()...)
	return migrations
}

func V1_9_3() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "20250213-v1.9.3-addMachineExtra-updateMenuIcon",
			Migrate: func(tx *gorm.DB) error {
				tx.Migrator().AddColumn(&machineentity.Machine{}, "extra")

				// 更新菜单图标
				resourceModel := &sysentity.Resource{}
				tx.Model(resourceModel).Where("id = ?", 11).Update("meta", `{"component":"system/role/RoleList","icon":"icon menu/role","isKeepAlive":true,"routeName":"RoleList"}`)
				tx.Model(resourceModel).Where("id = ?", 14).Update("meta", `{"component":"system/account/AccountList","icon":"User","isKeepAlive":true,"routeName":"AccountList"}`)
				tx.Model(resourceModel).Where("id = ?", 150).Update("meta", `{"component":"ops/db/SyncTaskList","icon":"Refresh","isKeepAlive":true,"routeName":"SyncTaskList"}`)
				tx.Model(resourceModel).Where("id = ?", 60).Update("meta", `{"icon":"icon redis/redis","isKeepAlive":true,"routeName":"RDS"}`)
				tx.Model(resourceModel).Where("id = ?", 61).Update("meta", `{"component":"ops/redis/DataOperation","icon":"icon redis/redis","isKeepAlive":true,"routeName":"DataOperation"}`)
				tx.Model(resourceModel).Where("id = ?", 63).Update("meta", `{"component":"ops/redis/RedisList","icon":"icon redis/redis","isKeepAlive":true,"routeName":"RedisList"}`)
				tx.Model(resourceModel).Where("id = ?", 79).Update("meta", `{"icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"Mongo"}`)
				tx.Model(resourceModel).Where("id = ?", 80).Update("meta", `{"component":"ops/mongo/MongoDataOp","icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"MongoDataOp"}`)
				tx.Model(resourceModel).Where("id = ?", 82).Update("meta", `{"component":"ops/mongo/MongoList","icon":"icon mongo/mongo","isKeepAlive":true,"routeName":"MongoList"}`)
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}
