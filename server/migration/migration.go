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

	err := run(db,
		migrations.Init,
		migrations.V1_9,
		migrations.V1_10,
	)

	if err == nil {
		logx.Info("migrations run success")
	}
	return err
}

func run(db *gorm.DB, fs ...func() []*gormigrate.Migration) error {
	var ms []*gormigrate.Migration
	for _, f := range fs {
		ms = append(ms, f()...)
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
