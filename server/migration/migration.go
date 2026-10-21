package migration

import (
	"mayfly-go/migration/migrations"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/rediscli"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// RunMigrations 数据库迁移操作
func RunMigrations(db *gorm.DB) error {
	logx.Info("start to run migrations")
	// 添加分布式锁, 防止多个服务同时执行迁移
	lock := rediscli.NewLock("mayfly:db:migrations", 1*time.Minute)
	if lock != nil {
		if !lock.Lock() {
			return nil
		}
		defer lock.UnLock()
	}

	// 迁移链的顺序与批次清单以 migrations.All() 为唯一真源，本文件不再逐个点名
	err := run(db, migrations.All()...)

	if err == nil {
		logx.Info("migrations run success")
	}
	return err
}

func run(db *gorm.DB, batches ...migrations.Batch) error {
	var ms []*gormigrate.Migration
	for _, batch := range batches {
		ms = append(ms, batch()...)
	}

	m := gormigrate.New(db, &gormigrate.Options{
		TableName:                 "migrations",
		IDColumnName:              "id",
		IDColumnSize:              300,
		UseTransaction:            true,
		ValidateUnknownMigrations: false,
	}, ms)
	if err := m.Migrate(); err != nil {
		return err
	}
	return nil
}
