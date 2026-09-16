package migrations

import (
	"encoding/json"
	"errors"
	"strings"

	aientity "mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/skill"
	alertentity "mayfly-go/internal/alert/domain/entity"
	dbentity "mayfly-go/internal/db/domain/entity"
	fileentity "mayfly-go/internal/file/domain/entity"
	labelentity "mayfly-go/internal/label/domain/entity"
	machineentity "mayfly-go/internal/machine/domain/entity"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// V1_12 v1.12.0 全量迁移，按领域分节组织。
// 所有迁移均幂等设计：重复执行安全（按 id / code / column 查重后跳过）。
func V1_12() []*gormigrate.Migration {
	return []*gormigrate.Migration{

		// ================================================================
		//  AI 模块
		// ================================================================

		{
			// 对话 / 轮次新模型建表（Conversation + TurnItem）
			ID: "v1.12.0-ai-chat-refactor",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&aientity.Conversation{}, &aientity.TurnItem{})
			},
			Rollback: noopRollback,
		},
		{
			// AI 菜单角色权限补齐（V1_11 遗漏），按 (role_id, resource_id) 查重
			ID: "v1.12.0-ai-menu-permission",
			Migrate: func(tx *gorm.DB) error {
				for _, rid := range []int64{1775967861, 1775967903} {
					if err := grantRoleResource(tx, 1, rid); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: noopRollback,
		},
		{
			// LLM 消息并入 TurnItem 后，移除废弃的 session / session_message 表
			ID: "v1.12.0-ai-session-merge-turn-item",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE IF EXISTS t_ai_session, t_ai_session_message").Error
			},
			Rollback: noopRollback,
		},
		{
			// TurnItem: payload 为唯一事实源，移除 action_id 列
			ID: "v1.12.0-ai-turn-item-drop-action-id",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&aientity.TurnItem{}); err != nil {
					return err
				}
				if tx.Migrator().HasColumn(&aientity.TurnItem{}, "action_id") {
					return tx.Migrator().DropColumn(&aientity.TurnItem{}, "action_id")
				}
				return nil
			},
			Rollback: noopRollback,
		},
		{
			// 长期记忆落库（t_ai_memory），替代本地 JSONL，跨实例共享
			ID: "v1.12.0-ai-memory-table",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&aientity.Memory{})
			},
			Rollback: noopRollback,
		},
		{
			// AI 模型配置补齐 maxTokens / enableThinking 可配置项
			ID: "v1.12.0-ai-model-config-params",
			Migrate: func(tx *gorm.DB) error {
				return appendSysConfigFields(tx, "AiModelConfig", []map[string]any{
					{"model": "maxTokens", "name": "system.sysconf.aiMaxTokens", "placeholder": "system.sysconf.aiMaxTokensPlaceholder", "required": false},
					{"model": "enableThinking", "name": "system.sysconf.aiEnableThinking", "options": "true,false", "required": false},
				}, func(tx *gorm.DB, config *sysentity.Config) error {
					// value 补写默认值：显式开启思考模式
					var val map[string]any
					if err := json.Unmarshal([]byte(config.Value), &val); err != nil {
						return err
					}
					if _, ok := val["enableThinking"]; !ok {
						val["enableThinking"] = "true"
						value, err := json.Marshal(val)
						if err != nil {
							return err
						}
						return tx.Model(config).Update("value", string(value)).Error
					}
					return nil
				})
			},
			Rollback: noopRollback,
		},
		{
			// 全局 AI 助手悬浮球权限码（ai:chat），挂在 AI 助手菜单下
			ID: "v1.12.0-ai-chat-permission",
			Migrate: func(tx *gorm.DB) error {
				return insertSysResourceWithRole(tx, 1775967920, 1775967903, "lUgXhO96/VPfzI5pQ/Qk7wRm3x/",
					2, "menu.aiChat", "ai:chat", 1775967920, "null")
			},
			Rollback: noopRollback,
		},
		{
			// AI 集成插件体系：技能两表 + 内置技能落库 + AI 集成菜单
			ID: "v1.12.0-ai-plugin-integration",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&aientity.Skill{}, &aientity.SkillResource{}); err != nil {
					return err
				}
				if err := seedBuiltinSkills(tx); err != nil {
					return err
				}
				return insertSysResourcesWithRole(tx, []sysResourceDef{
					{1775967910, 1775967861, "lUgXhO96/V3rTnK8w/", 1, "menu.aiIntegration", "integration", 1775967910, `{"icon":"icon ai/plugin","isKeepAlive":true}`},
					{1775967911, 1775967910, "lUgXhO96/V3rTnK8w/Q9mPy6d2/", 1, "menu.aiPluginManagement", "plugin", 1775967911, `{"icon":"icon ai/plugin","isKeepAlive":true,"routeName":"AiPluginManagement"}`},
				})
			},
			Rollback: noopRollback,
		},
		{
			// 插件实例统一视图（t_ai_plugin_instance），存量技能同步为实例
			ID: "v1.12.0-ai-plugin-instance",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&aientity.PluginInstance{}); err != nil {
					return err
				}
				return tx.Exec(
					"INSERT INTO t_ai_plugin_instance (code, plugin_type, name, description, config, enabled, status, creator_id, creator, modifier_id, modifier, create_time, update_time) " +
						"SELECT s.code, 'skill', s.name, s.description, CONCAT('{\"skillCode\":\"', s.code, '\"}'), 1, 0, 1, 'admin', 1, 'admin', NOW(), NOW() " +
						"FROM t_ai_skill s WHERE NOT EXISTS (SELECT 1 FROM t_ai_plugin_instance i WHERE i.code = s.code)").Error
			},
			Rollback: noopRollback,
		},
		{
			// AI 设置页面菜单（专用配置表单：failover / toolSearchThreshold）
			ID: "v1.12.0-ai-settings-menu",
			Migrate: func(tx *gorm.DB) error {
				return insertSysResourceWithRole(tx, 1775967921, 1775967861, "lUgXhO96/Ks3wQ7mN/",
					1, "menu.aiSettings", "settings", 1775967921, `{"icon":"icon ai/ai","isKeepAlive":true,"routeName":"AiSettings"}`)
			},
			Rollback: noopRollback,
		},

		// ================================================================
		//  文件 / 系统配置
		// ================================================================

		{
			// 文件配置增加 S3 对象存储表单项
			ID: "v1.12.0-file-config-s3-params",
			Migrate: func(tx *gorm.DB) error {
				return appendSysConfigFields(tx, "FileConfig", []map[string]any{
					{"model": "s3Endpoint", "name": "system.sysconf.s3Endpoint", "placeholder": "system.sysconf.s3EndpointPlaceholder", "required": false},
					{"model": "s3Region", "name": "system.sysconf.s3Region", "placeholder": "system.sysconf.s3RegionPlaceholder", "required": false},
					{"model": "s3Bucket", "name": "system.sysconf.s3Bucket", "placeholder": "system.sysconf.s3BucketPlaceholder", "required": false},
					{"model": "s3AccessKey", "name": "system.sysconf.s3AccessKey", "placeholder": "system.sysconf.s3AccessKeyPlaceholder", "required": false},
					{"model": "s3SecretKey", "name": "system.sysconf.s3SecretKey", "placeholder": "system.sysconf.s3SecretKeyPlaceholder", "required": false},
					{"model": "s3PathStyle", "name": "system.sysconf.s3PathStyle", "placeholder": "system.sysconf.s3PathStylePlaceholder", "options": "true,false", "required": false},
				}, func(tx *gorm.DB, _ *sysentity.Config) error {
					// 直接更新 DB 绕过应用层缓存失效逻辑，需主动清除
					cache.Del(sysapp.SysConfigKeyPrefix + "FileConfig")
					return nil
				})
			},
			Rollback: noopRollback,
		},
		{
			// 文件表增加存储介质（storage_type）与元信息（extra）字段
			ID: "v1.12.0-file-storage-type",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&fileentity.File{})
			},
			Rollback: noopRollback,
		},
		{
			// 动态表单 params 升级为 v1 JSON Schema（旧版裸数组 → {version:1, fields:[...]}）
			ID: "v1.12.0-form-params-json-schema-v1",
			Migrate: func(tx *gorm.DB) error {
				return migrateFormParamsToJsonSchema(tx)
			},
			Rollback: noopRollback,
		},
		{
			// DbmsConfig 补充脱敏配置项（maskEnabled / maskFailClosed / maskExemptRoleIds）
			ID: "v1.12.0-dbms-config-mask-items",
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
			Rollback: noopRollback,
		},

		// ================================================================
		//  数据库管理增强（脱敏 / 迁移任务并发 & 断点续传）
		// ================================================================

		{
			// 数据库字段脱敏：规则库 + 列标签表 + 内置脱敏规则 + 菜单
			ID: "v1.12.0-db-field-mask",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&dbentity.DbMaskRule{}, &dbentity.DbMaskColumn{}); err != nil {
					return err
				}
				// 内置脱敏规则（按 id 查重）
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
					if err := tx.Exec(
						"INSERT INTO t_db_mask_rule (id, name, match_type, pattern, algorithm, params, status, weight, remark, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) "+
							"VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, 1, 'admin', 1, 'admin', NOW(), NOW(), 0, NULL)",
						r.id, r.name, r.matchType, r.pattern, r.algorithm, r.params, r.weight, "内置规则",
					).Error; err != nil {
						return err
					}
				}
				// 脱敏规则菜单 + 角色权限
				return insertSysResourceWithRole(tx, 1775967930, 36, "dbms23ax/Ma3kRu1ex/",
					1, "menu.dbMaskRule", "db-mask", 10000000,
					`{"component":"ops/db/component/mask/MaskRuleList","icon":"Coin","isKeepAlive":true,"routeName":"DbMaskRuleList"}`)
			},
			Rollback: noopRollback,
		},
		{
			// DB 迁移任务增强：并发度列 + 断点续传检查点表
			ID: "v1.12.0-db-transfer-concurrency-checkpoint",
			Migrate: func(tx *gorm.DB) error {
				if !tx.Migrator().HasColumn(&dbentity.DbTransferTask{}, "concurrency") {
					if err := tx.Migrator().AddColumn(&dbentity.DbTransferTask{}, "concurrency"); err != nil {
						return err
					}
				}
				if !tx.Migrator().HasTable(&dbentity.DbTransferCheckpoint{}) {
					if err := tx.AutoMigrate(&dbentity.DbTransferCheckpoint{}); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: noopRollback,
		},

		// ================================================================
		//  告警模块
		// ================================================================

		{
			// 告警模块全部业务表（含通知日志表）
			ID: "v1.12.0-alert-tables",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(
					new(alertentity.AlertRule),
					new(alertentity.AlertEvent),
					new(alertentity.AlertSilence),
					new(alertentity.AlertEscalation),
					new(alertentity.AlertInhibition),
					new(alertentity.AlertNotifyPolicy),
					new(alertentity.AlertNotifyLog),
				)
			},
			Rollback: noopRollback,
		},
		{
			// 告警模块全部资源（菜单 + 按钮权限）+ 角色权限授予
			ID: "v1.12.0-alert-resources",
			Migrate: func(tx *gorm.DB) error {
				return insertSysResourcesWithRole(tx, alertResources)
			},
			Rollback: noopRollback,
		},

		// ================================================================
		//  全局标签管理
		// ================================================================

		{
			// 标签注册表 + 绑定表
			ID: "v1.12.0-label-tables",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(new(labelentity.Label), new(labelentity.LabelBinding))
			},
			Rollback: noopRollback,
		},
		{
			// 标签管理菜单 + 按钮权限 + 角色权限
			ID: "v1.12.0-label-resources",
			Migrate: func(tx *gorm.DB) error {
				return insertSysResourcesWithRole(tx, labelResources)
			},
			Rollback: noopRollback,
		},
	}
}

// ================================================================
//  公共辅助函数
// ================================================================

// noopRollback 空回滚（迁移不支持回滚时统一使用）
func noopRollback(tx *gorm.DB) error { return nil }

// sysResourceDef 系统资源定义（菜单 / 按钮权限通用）
type sysResourceDef struct {
	id     int64
	pid    int64
	uiPath string
	typ    int8
	name   string
	code   string
	weight int
	meta   string
}

// insertSysResource 按 id 查重后插入单条系统资源，已存在则跳过
func insertSysResource(tx *gorm.DB, r sysResourceDef) error {
	var count int64
	if err := tx.Table("t_sys_resource").Where("id = ? AND is_deleted = 0", r.id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Exec(
		"INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) "+
			"VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, 1, 'admin', 1, 'admin', NOW(), NOW(), 0, NULL)",
		r.id, r.pid, r.uiPath, r.typ, r.name, r.code, r.weight, r.meta,
	).Error
}

// insertSysResourcesWithRole 批量插入资源并授予 COMMON 角色（role_id=1）
func insertSysResourcesWithRole(tx *gorm.DB, resources []sysResourceDef) error {
	for _, r := range resources {
		if err := insertSysResource(tx, r); err != nil {
			return err
		}
		if err := grantRoleResource(tx, 1, r.id); err != nil {
			return err
		}
	}
	return nil
}

// grantRoleResource 按 (role_id, resource_id) 查重后授予角色资源权限
func grantRoleResource(tx *gorm.DB, roleId, resourceId int64) error {
	var count int64
	if err := tx.Table("t_sys_role_resource").Where("role_id = ? AND resource_id = ? AND is_deleted = 0", roleId, resourceId).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Exec(
		"INSERT INTO t_sys_role_resource (role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (?, ?, 1, 'admin', NOW(), 0, NULL)",
		roleId, resourceId,
	).Error
}

// insertSysResourceWithRole 插入单条资源并授予 COMMON 角色
func insertSysResourceWithRole(tx *gorm.DB, id, pid int64, uiPath string, typ int8, name, code string, weight int, meta string) error {
	return insertSysResourcesWithRole(tx, []sysResourceDef{
		{id, pid, uiPath, typ, name, code, weight, meta},
	})
}

// appendSysConfigFields 向系统配置 params 追加新字段定义（按 model 去重）。
// afterUpdate 在 params 更新后执行（如清缓存、补默认值），无需后置操作时传 nil。
func appendSysConfigFields(tx *gorm.DB, configKey string, fields []map[string]any, afterUpdate func(*gorm.DB, *sysentity.Config) error) error {
	config := &sysentity.Config{}
	if err := tx.Where("`key` = ?", configKey).First(config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // 无该配置行则跳过
		}
		return err
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(config.Params), &items); err != nil {
		return err
	}
	changed := false
	for _, add := range fields {
		exists := false
		for _, it := range items {
			if it["model"] == add["model"] {
				exists = true
				break
			}
		}
		if !exists {
			items = append(items, add)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	params, err := json.Marshal(items)
	if err != nil {
		return err
	}
	if err := tx.Model(config).Update("params", string(params)).Error; err != nil {
		return err
	}
	if afterUpdate != nil {
		return afterUpdate(tx, config)
	}
	return nil
}

// seedBuiltinSkills 内置技能落库（按 code 查重，幂等）
func seedBuiltinSkills(tx *gorm.DB) error {
	for _, s := range builtinSkills {
		var count int64
		if err := tx.Model(&aientity.Skill{}).Where("code = ?", s.Code).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := tx.Exec(
			"INSERT INTO t_ai_skill (code, name, description, allowed_tools, version, source, status, creator_id, creator, modifier_id, modifier, create_time, update_time) "+
				"VALUES (?, ?, ?, '', '1.0.0', ?, ?, 1, 'admin', 1, 'admin', NOW(), NOW())",
			s.Code, s.Name, s.Description, aientity.SkillSourceBuiltin, aientity.SkillStatusPublished,
		).Error; err != nil {
			return err
		}
		var skillId uint64
		if err := tx.Raw("SELECT id FROM t_ai_skill WHERE code = ?", s.Code).Scan(&skillId).Error; err != nil {
			return err
		}
		content := skill.BuildSkillMd(s.Name, s.Description, "", strings.Join(s.Body, "\n"))
		if err := tx.Exec(
			"INSERT INTO t_ai_skill_resource (skill_id, path, content, size, creator_id, creator, modifier_id, modifier, create_time, update_time) "+
				"VALUES (?, ?, ?, ?, 1, 'admin', 1, 'admin', NOW(), NOW())",
			skillId, aientity.SkillMdPath, content, len(content),
		).Error; err != nil {
			return err
		}
	}
	return nil
}

// ================================================================
//  资源数据定义
// ================================================================

// alertResources 告警模块全部资源（菜单 + 按钮权限）
var alertResources = []sysResourceDef{
	// ── 菜单（Type=1）──
	{1730000001, 0, "Alert001/", 1, "menu.alert", "/alert", 49999997, `{"icon":"Bell","isKeepAlive":true,"routeName":"Alert"}`},
	{1730000002, 1730000001, "Alert001/Alrt0001/", 1, "menu.alertOverview", "alert-overview", 10000000, `{"component":"ops/alert/AlertOverview","icon":"DataLine","isKeepAlive":true,"routeName":"AlertOverview"}`},
	{1730000003, 1730000001, "Alert001/Alrt0002/", 1, "menu.alertRule", "alert-rules", 20000000, `{"component":"ops/alert/AlertRuleList","icon":"Setting","isKeepAlive":true,"routeName":"AlertRuleList"}`},
	{1730000004, 1730000001, "Alert001/Alrt0003/", 1, "menu.alertEvent", "alert-events", 30000000, `{"component":"ops/alert/AlertEventList","icon":"Warning","isKeepAlive":true,"routeName":"AlertEventList"}`},
	{1730000005, 1730000001, "Alert001/Alrt0004/", 1, "menu.alertSilence", "alert-silences", 40000000, `{"component":"ops/alert/AlertSilenceList","icon":"Mute","isKeepAlive":true,"routeName":"AlertSilenceList"}`},
	{1730000006, 1730000001, "Alert001/Alrt0005/", 1, "menu.alertEscalation", "alert-escalations", 50000000, `{"component":"ops/alert/AlertEscalationList","icon":"TopRight","isKeepAlive":true,"routeName":"AlertEscalationList"}`},
	{1730000007, 1730000001, "Alert001/Alrt0006/", 1, "menu.alertInhibition", "alert-inhibitions", 60000000, `{"component":"ops/alert/AlertInhibitionList","icon":"SwitchButton","isKeepAlive":true,"routeName":"AlertInhibitionList"}`},
	{1730000008, 1730000001, "Alert001/Alrt0007/", 1, "menu.alertNotifyPolicy", "alert-notify-policies", 70000000, `{"component":"ops/alert/AlertNotifyPolicyList","icon":"Message","isKeepAlive":true,"routeName":"AlertNotifyPolicyList"}`},
	// ── 规则按钮权限（Type=2）──
	{1730000010, 1730000003, "Alert001/Alrt0002/Rule0001/", 2, "menu.alertRuleBase", "alert:rule", 10000000, ""},
	{1730000011, 1730000003, "Alert001/Alrt0002/Rule0002/", 2, "menu.alertRuleSave", "alert:rule:save", 20000000, ""},
	{1730000012, 1730000003, "Alert001/Alrt0002/Rule0003/", 2, "menu.alertRuleDelete", "alert:rule:del", 30000000, ""},
	// ── 事件按钮权限 ──
	{1730000020, 1730000004, "Alert001/Alrt0003/Evt00001/", 2, "menu.alertEventBase", "alert:event", 10000000, ""},
	{1730000021, 1730000004, "Alert001/Alrt0003/Evt00002/", 2, "menu.alertEventAck", "alert:event:ack", 20000000, ""},
	// ── 抑制按钮权限 ──
	{1730000050, 1730000007, "Alert001/Alrt0006/Inh00001/", 2, "menu.alertInhibitionBase", "alert:inhibition", 10000000, ""},
	{1730000051, 1730000007, "Alert001/Alrt0006/Inh00002/", 2, "menu.alertInhibitionSave", "alert:inhibition:save", 20000000, ""},
	{1730000052, 1730000007, "Alert001/Alrt0006/Inh00003/", 2, "menu.alertInhibitionDelete", "alert:inhibition:del", 30000000, ""},
	// ── 升级按钮权限 ──
	{1730000040, 1730000006, "Alert001/Alrt0005/Esc00001/", 2, "menu.alertEscalationBase", "alert:escalation", 10000000, ""},
	{1730000041, 1730000006, "Alert001/Alrt0005/Esc00002/", 2, "menu.alertEscalationSave", "alert:escalation:save", 20000000, ""},
	{1730000042, 1730000006, "Alert001/Alrt0005/Esc00003/", 2, "menu.alertEscalationDelete", "alert:escalation:del", 30000000, ""},
	// ── 静默按钮权限 ──
	{1730000030, 1730000005, "Alert001/Alrt0004/Sil00001/", 2, "menu.alertSilenceBase", "alert:silence", 10000000, ""},
	{1730000031, 1730000005, "Alert001/Alrt0004/Sil00002/", 2, "menu.alertSilenceSave", "alert:silence:save", 20000000, ""},
	{1730000032, 1730000005, "Alert001/Alrt0004/Sil00003/", 2, "menu.alertSilenceDelete", "alert:silence:del", 30000000, ""},
	// ── 通知策略按钮权限 ──
	{1730000060, 1730000008, "Alert001/Alrt0007/Np00001/", 2, "menu.alertNotifyPolicyBase", "alert:notifyPolicy", 10000000, ""},
	{1730000061, 1730000008, "Alert001/Alrt0007/Np00002/", 2, "menu.alertNotifyPolicySave", "alert:notifyPolicy:save", 20000000, ""},
	{1730000062, 1730000008, "Alert001/Alrt0007/Np00003/", 2, "menu.alertNotifyPolicyDelete", "alert:notifyPolicy:del", 30000000, ""},
}

// labelResources 标签管理模块资源（挂在「资源」一级菜单 id=93 下）
var labelResources = []sysResourceDef{
	{1740000001, 93, "Tag3fhad/Lbl00001/", 1, "menu.label", "label-list", 15000000, `{"component":"ops/label/LabelList","icon":"PriceTag","isKeepAlive":true,"routeName":"LabelList"}`},
	{1740000010, 1740000001, "Tag3fhad/Lbl00001/Lbl00002/", 2, "menu.labelBase", "label:base", 10000000, ""},
	{1740000011, 1740000001, "Tag3fhad/Lbl00001/Lbl00003/", 2, "menu.labelSave", "label:save", 20000000, ""},
	{1740000012, 1740000001, "Tag3fhad/Lbl00001/Lbl00004/", 2, "menu.labelDelete", "label:del", 30000000, ""},
}

// ================================================================
//  动态表单 params 升级 v1 JSON Schema
// ================================================================

// legacyFormParam 旧版动态表单字段定义（params 为裸数组，元素含 model 字段）
type legacyFormParam struct {
	Model       string `json:"model"`
	Name        string `json:"name"`
	Placeholder string `json:"placeholder"`
	Options     string `json:"options"`
	Required    bool   `json:"required"`
}

type jsonFormOption struct {
	Value any    `json:"value"`
	Label string `json:"label"`
}

type jsonFormRules struct {
	Required bool `json:"required,omitempty"`
}

type jsonFormField struct {
	Prop        string           `json:"prop"`
	Label       string           `json:"label,omitempty"`
	Type        string           `json:"type,omitempty"`
	Placeholder string           `json:"placeholder,omitempty"`
	Rules       *jsonFormRules   `json:"rules,omitempty"`
	Options     []jsonFormOption `json:"options,omitempty"`
}

type jsonFormSchema struct {
	Version int             `json:"version"`
	Fields  []jsonFormField `json:"fields"`
}

// convertLegacyFormParams 将旧版动态表单字段定义 JSON 转换为 v1 Schema JSON。
// 入参为空、非 JSON、JSON 对象（已是 v1）时返回 "" 表示无需转换。
func convertLegacyFormParams(params string) (string, error) {
	trimmed := strings.TrimSpace(params)
	if trimmed == "" {
		return "", nil
	}
	// JSON 对象（含 version 的 v1 Schema）跳过，保证幂等
	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &probe); err == nil {
		return "", nil
	}
	// 非 JSON 数组（旧格式之外的异常数据）不处理
	var legacy []legacyFormParam
	if err := json.Unmarshal([]byte(trimmed), &legacy); err != nil {
		return "", nil
	}
	schema := jsonFormSchema{Version: 1, Fields: make([]jsonFormField, 0, len(legacy))}
	for _, p := range legacy {
		field := jsonFormField{
			Prop:        p.Model,
			Label:       p.Name,
			Placeholder: p.Placeholder,
		}
		if p.Required {
			field.Rules = &jsonFormRules{Required: true}
		}
		if p.Options != "" {
			for _, option := range strings.Split(p.Options, ",") {
				option = strings.TrimSpace(option)
				if option == "" {
					continue
				}
				field.Options = append(field.Options, jsonFormOption{Value: option, Label: option})
			}
		}
		schema.Fields = append(schema.Fields, field)
	}
	converted, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}
	return string(converted), nil
}

// migrateFormParamsToJsonSchema 存量动态表单定义升级：t_sys_config.params 与 machine_scripts.params
func migrateFormParamsToJsonSchema(tx *gorm.DB) error {
	// 转换后 JSON 膨胀约 20%，先扩列（t_sys_config varchar 1500→4000，machine_scripts 500→1500）
	if err := tx.AutoMigrate(&sysentity.Config{}, &machineentity.MachineScript{}); err != nil {
		return err
	}
	// t_sys_config：逐行转换，更新后失效对应配置缓存
	var configs []sysentity.Config
	if err := tx.Find(&configs).Error; err != nil {
		return err
	}
	for _, config := range configs {
		converted, err := convertLegacyFormParams(config.Params)
		if err != nil {
			logx.Errorf("convert config form params to json schema failed, key: %s, err: %v", config.Key, err)
			continue
		}
		if converted == "" {
			continue
		}
		if err := tx.Model(&sysentity.Config{}).Where("id = ?", config.Id).Update("params", converted).Error; err != nil {
			return err
		}
		cache.Del(sysapp.SysConfigKeyPrefix + config.Key)
	}
	// t_machine_script：脚本入参定义，无缓存依赖
	var scripts []machineentity.MachineScript
	if err := tx.Find(&scripts).Error; err != nil {
		return err
	}
	for _, script := range scripts {
		converted, err := convertLegacyFormParams(script.Params)
		if err != nil {
			logx.Errorf("convert script form params to json schema failed, id: %d, err: %v", script.Id, err)
			continue
		}
		if converted == "" {
			continue
		}
		if err := tx.Model(&machineentity.MachineScript{}).Where("id = ?", script.Id).Update("params", converted).Error; err != nil {
			return err
		}
	}
	return nil
}
