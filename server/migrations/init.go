package migrations

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/rediscli"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// RunMigrations 数据库迁移操作
func RunMigrations(db *gorm.DB) error {
	// 添加分布式锁, 防止多个服务同时执行迁移
	lock := rediscli.NewLock("mayfly:db:migrations", 1*time.Minute)
	if lock != nil {
		if !lock.Lock() {
			return nil
		}
		defer lock.UnLock()
	}

	if !config.Conf.Mysql.AutoMigration {
		return nil
	}

	return run(db,
		// T2022,
		// T20230720,
		T20231125,
	)
}

func run(db *gorm.DB, fs ...func() *gormigrate.Migration) error {
	var ms []*gormigrate.Migration
	for _, f := range fs {
		ms = append(ms, f())
	}
	m := gormigrate.New(db, &gormigrate.Options{
		TableName:                 "migrations",
		IDColumnName:              "id",
		IDColumnSize:              200,
		UseTransaction:            true,
		ValidateUnknownMigrations: true,
	}, ms)
	if err := m.Migrate(); err != nil {
		return err
	}
	return nil
}

func insertResource(tx *gorm.DB, res *entity.Resource) error {
	now := time.Now()
	res.CreateTime = &now
	res.CreatorId = 1
	res.Creator = "admin"
	res.UpdateTime = &now
	res.ModifierId = 1
	res.Modifier = "admin"
	if err := tx.Save(res).Error; err != nil {
		return err
	}
	return tx.Save(&entity.RoleResource{
		DeletedModel: model.DeletedModel{},
		RoleId:       1,
		ResourceId:   res.Id,
		CreateTime:   &now,
		CreatorId:    1,
		Creator:      "admin",
	}).Error
}
