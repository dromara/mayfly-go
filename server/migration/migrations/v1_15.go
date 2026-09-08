package migrations

import (
	sysentity "mayfly-go/internal/sys/domain/entity"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func V1_15() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			// 系统配置 DbmsConfig 补充脱敏配置项（maskEnabled/maskFailClosed/maskExemptRoleIds），
			// 并将旧版数组格式配置项定义升级为 v1 JSON schema（配置弹窗渲染为表单项）。
			// 背景：旧版定义无脱敏配置项，maskEnabled 默认关闭且界面无法开启，
			// 导致已配置的脱敏规则/列标签在查询时不生效。
			// 幂等：params 已含脱敏配置（maskEnabled）时跳过
			ID: "v1.15.0-dbms-config-mask-items",
			Migrate: func(tx *gorm.DB) error {
				var count int64
				if err := tx.Model(&sysentity.Config{}).
					Where(&sysentity.Config{Key: "DbmsConfig"}).
					Where("params LIKE ?", "%maskEnabled%").
					Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return nil
				}
				return tx.Model(&sysentity.Config{}).
					Where(&sysentity.Config{Key: "DbmsConfig"}).
					Update("params", dbmsConfParams).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}
