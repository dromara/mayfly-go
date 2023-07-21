package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/model"
	"time"
)

// T20230720 三方登录表
func T20230720() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20230319",
		Migrate: func(tx *gorm.DB) error {
			// 添加路由权限
			now := time.Now()
			res := &entity.Resource{
				Model: model.Model{
					DeletedModel: model.DeletedModel{Id: 130},
					CreateTime:   &now,
					CreatorId:    1,
					Creator:      "admin",
					UpdateTime:   &now,
					ModifierId:   1,
					Modifier:     "admin",
				},
				Pid:    4,
				UiPath: "sys/auth",
				Type:   1,
				Status: 1,
				Code:   "system:auth",
				Name:   "登录认证",
				Weight: 10000001,
				Meta: "{\"component\":\"system/auth/AuthInfo\"," +
					"\"icon\":\"User\",\"isKeepAlive\":true," +
					"\"routeName\":\"AuthInfo\"}",
			}
			if err := tx.Save(res).Error; err != nil {
				return err
			}
			res = &entity.Resource{
				Model: model.Model{
					DeletedModel: model.DeletedModel{Id: 131},
					CreateTime:   &now,
					CreatorId:    1,
					Creator:      "admin",
					UpdateTime:   &now,
					ModifierId:   1,
					Modifier:     "admin",
				},
				Pid:    130,
				UiPath: "sys/auth/base",
				Type:   2,
				Status: 1,
				Code:   "system:auth:base",
				Name:   "基本权限",
				Weight: 10000000,
				Meta:   "null",
			}
			if err := tx.Save(res).Error; err != nil {
				return err
			}
			// 加大params字段长度
			if err := tx.Exec("alter table " + (&entity.Config{}).TableName() +
				" modify column params varchar(1000)").Error; err != nil {
				return err
			}
			if err := tx.Save(&entity.Config{
				Model: model.Model{
					CreateTime: &now,
					CreatorId:  1,
					Creator:    "admin",
					UpdateTime: &now,
					ModifierId: 1,
					Modifier:   "admin",
				},
				Name:   api.AuthOAuth2Name,
				Key:    api.AuthOAuth2Key,
				Params: api.AuthOAuth2Param,
				Value:  "{}",
				Remark: api.AuthOAuth2Remark,
			}).Error; err != nil {
				return err
			}
			return tx.AutoMigrate(&entity.OAuthAccount{})
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
