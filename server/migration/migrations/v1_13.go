package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	dbentity "mayfly-go/internal/db/domain/entity"
)

func V1_13() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			// 数据库查询结果字段脱敏：规则库（t_db_mask_rule）+ 列标签（t_db_mask_column）
			// AutoMigrate 幂等；内置规则与菜单按 id 查重后插入，重复执行安全
			ID: "v1.13.0-db-field-mask",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&dbentity.DbMaskRule{}, &dbentity.DbMaskColumn{}); err != nil {
					return err
				}

				// 内置常用脱敏规则（参考 Archery/Bytebase 预设），权重越大越先匹配
				seedRules := []struct {
					id        uint64
					name      string
					matchType int8
					pattern   string
					algorithm string
					params    string
					weight    int
				}{
					{1780000001, "手机号", dbentity.MaskMatchTypeRegex, `(?i)(mobile|phone|tel)$`, "phone", "", 100},
					{1780000002, "邮箱", dbentity.MaskMatchTypeRegex, `(?i)e?mail$`, "email", "", 100},
					{1780000003, "身份证", dbentity.MaskMatchTypeRegex, `(?i)(id_?card|identity_?no|id_?number|idcard)$`, "idcard", "", 100},
					{1780000004, "银行卡", dbentity.MaskMatchTypeRegex, `(?i)(bank_?card|card_?no)$`, "bankCard", "", 100},
					{1780000005, "姓名", dbentity.MaskMatchTypeRegex, `(?i)(real_?name|nick_?name|user_?name)$`, "full", "", 90},
					{1780000006, "地址", dbentity.MaskMatchTypeRegex, `(?i)address$`, "partial", `{"keepFirst":2}`, 90},
					{1780000007, "联系方式-前缀", dbentity.MaskMatchTypePrefix, `contact_`, "phone", "", 80},
					{1780000008, "密钥令牌", dbentity.MaskMatchTypeRegex, `(?i)(secret|token|api_?key)$`, "full", "", 110},
				}
				for _, r := range seedRules {
					var count int64
					if err := tx.Table("t_db_mask_rule").Where("id = ? AND is_deleted = 0", r.id).Count(&count).Error; err != nil {
						return err
					}
					if count > 0 {
						continue
					}
					if err := tx.Exec("INSERT INTO t_db_mask_rule (id, name, match_type, pattern, algorithm, params, status, weight, remark, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) "+
						"VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, 1, 'admin', 1, 'admin', NOW(), NOW(), 0, NULL)",
						r.id, r.name, r.matchType, r.pattern, r.algorithm, r.params, r.weight, "内置规则").Error; err != nil {
						return err
					}
				}

				// 菜单：数据库管理目录（id=36）下插入「数据脱敏」页面（按 id 查重后插入）
				var menuCount int64
				if err := tx.Table("t_sys_resource").Where("id = ? AND is_deleted = 0", 1775967930).Count(&menuCount).Error; err != nil {
					return err
				}
				if menuCount == 0 {
					if err := tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) "+
						"VALUES (?, ?, ?, 1, 1, ?, ?, ?, ?, 1, 'admin', 1, 'admin', NOW(), NOW(), 0, NULL)",
						1775967930, 36, "dbms23ax/Ma3kRu1ex/", "menu.dbMaskRule", "db-mask", 10000000,
						`{"component":"ops/db/component/mask/MaskRuleList","icon":"Coin","isKeepAlive":true,"routeName":"DbMaskRuleList"}`).Error; err != nil {
						return err
					}
					// 角色权限同步授予（与 v1.12 菜单迁移同模式）
					if err := tx.Exec("INSERT INTO t_sys_role_resource (role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (1, ?, 1, 'admin', NOW(), 0, NULL)", 1775967930).Error; err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}
