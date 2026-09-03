package migrations

import (
	"encoding/json"
	"errors"
	"strings"

	aientity "mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/skill"
	fileentity "mayfly-go/internal/file/domain/entity"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/cache"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func V1_12() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "v1.12.0-ai-chat-refactor",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(
					&aientity.Conversation{},
					&aientity.TurnItem{},
				)
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "v1.12.0-ai-menu-permission",
			Migrate: func(tx *gorm.DB) error {
				// 补充 AI 菜单角色权限（V1_11 遗漏）；已存在则跳过，保证重复执行安全
				for _, resourceId := range []int64{1775967861, 1775967903} {
					var count int64
					if err := tx.Table("t_sys_role_resource").Where("role_id = ? AND resource_id = ? AND is_deleted = 0", 1, resourceId).Count(&count).Error; err != nil {
						return err
					}
					if count > 0 {
						continue
					}
					if err := tx.Exec("INSERT INTO t_sys_role_resource (role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (?, ?, 1, 'admin', NOW(), 0, NULL)", 1, resourceId).Error; err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			// LLM 消息并入 turn_item 存储后移除独立 session 表：
			// 消息已由 api 层以 TurnItem 形式持久化，t_ai_session(_message) 不再使用
			ID: "v1.12.0-ai-session-merge-turn-item",
			Migrate: func(tx *gorm.DB) error {
				// 移除旧 session 表（IF EXISTS 兼容未建过表的库）
				return tx.Exec("DROP TABLE IF EXISTS t_ai_session, t_ai_session_message").Error
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			// 对齐 tokhub TurnItemRow 模型：payload 为唯一事实源，仅 tool_call_id 建查询列；
			// 移除 action_id 列（中断定位信息自含于 payload 的 internal.extra.interruptEvent），
			// extra 由 string JSON 文本改为嵌套 ExtraData（gorm 序列化 JSON 对象）
			ID: "v1.12.0-ai-turn-item-drop-action-id",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&aientity.TurnItem{}); err != nil {
					return err
				}
				// 条件删除 action_id 列（老库可能已由旧版迁移补列）
				if tx.Migrator().HasColumn(&aientity.TurnItem{}, "action_id") {
					if err := tx.Migrator().DropColumn(&aientity.TurnItem{}, "action_id"); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			// 长期记忆落库（t_ai_memory）：记忆由本地 JSONL 迁移为 DB 存储，
			// 跨会话、跨实例共享（多实例部署无状态化收口）
			ID: "v1.12.0-ai-memory-table",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(
					&aientity.Memory{},
				)
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			// AI 模型配置补齐 maxTokens / enableThinking 可配置项：
			// thinking 模型的 reasoning 计入 max_tokens 输出预算（默认不限制，
			// 需控成本时才配置）；enableThinking 默认开启（对齐 qwen3 服务端
			// 流式默认行为），若遇工具调用不稳定可显式配置为 false
			ID: "v1.12.0-ai-model-config-params",
			Migrate: func(tx *gorm.DB) error {
				config := &sysentity.Config{}
				if err := tx.Where("`key` = ?", "AiModelConfig").First(config).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// 无该配置行则跳过（首次创建时已携带新字段定义与默认值）
						return nil
					}
					return err
				}
				var items []map[string]any
				if err := json.Unmarshal([]byte(config.Params), &items); err != nil {
					return err
				}
				appendItems := []map[string]any{
					{"model": "maxTokens", "name": "system.sysconf.aiMaxTokens", "placeholder": "system.sysconf.aiMaxTokensPlaceholder", "required": false},
					{"model": "enableThinking", "name": "system.sysconf.aiEnableThinking", "options": "true,false", "required": false},
				}
				for _, add := range appendItems {
					exists := false
					for _, it := range items {
						if it["model"] == add["model"] {
							exists = true
							break
						}
					}
					if !exists {
						items = append(items, add)
					}
				}
				params, err := json.Marshal(items)
				if err != nil {
					return err
				}
				if err := tx.Model(config).Update("params", string(params)).Error; err != nil {
					return err
				}
				// value 补写默认值：显式开启思考模式，配置界面可见可改
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
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			// 文件配置增加s3对象存储表单项（存量环境的FileConfig记录需补充params才能在配置界面展示s3字段）
			ID: "v1.12.0-file-config-s3-params",
			Migrate: func(tx *gorm.DB) error {
				config := &sysentity.Config{}
				if err := tx.Where("`key` = ?", "FileConfig").First(config).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// 无该配置行则跳过（首次创建时已携带新字段定义）
						return nil
					}
					return err
				}
				var items []map[string]any
				if err := json.Unmarshal([]byte(config.Params), &items); err != nil {
					return err
				}
				appendItems := []map[string]any{
					{"model": "s3Endpoint", "name": "system.sysconf.s3Endpoint", "placeholder": "system.sysconf.s3EndpointPlaceholder", "required": false},
					{"model": "s3Region", "name": "system.sysconf.s3Region", "placeholder": "system.sysconf.s3RegionPlaceholder", "required": false},
					{"model": "s3Bucket", "name": "system.sysconf.s3Bucket", "placeholder": "system.sysconf.s3BucketPlaceholder", "required": false},
					{"model": "s3AccessKey", "name": "system.sysconf.s3AccessKey", "placeholder": "system.sysconf.s3AccessKeyPlaceholder", "required": false},
					{"model": "s3SecretKey", "name": "system.sysconf.s3SecretKey", "placeholder": "system.sysconf.s3SecretKeyPlaceholder", "required": false},
					{"model": "s3PathStyle", "name": "system.sysconf.s3PathStyle", "placeholder": "system.sysconf.s3PathStylePlaceholder", "options": "true,false", "required": false},
				}
				for _, add := range appendItems {
					exists := false
					for _, it := range items {
						if it["model"] == add["model"] {
							exists = true
							break
						}
					}
					if !exists {
						items = append(items, add)
					}
				}
				params, err := json.Marshal(items)
				if err != nil {
					return err
				}
				if err := tx.Model(config).Update("params", string(params)).Error; err != nil {
					return err
				}
				// 直接更新db绕过了配置应用层的缓存失效逻辑，需主动清除避免读到旧缓存
				cache.Del(sysapp.SysConfigKeyPrefix + "FileConfig")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			// 文件表增加存储介质与元信息字段：storage_type区分local/s3，
			// 介质切换后仍按文件记录的原介质读写存量文件（存量数据默认local）；
			// extra记录s3 bucket元信息，bucket变更后访问文件时给出明确错误而非莫名not found
			ID: "v1.12.0-file-storage-type",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&fileentity.File{})
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			// AI 集成插件体系：技能（t_ai_skill + t_ai_skill_resource）与
			// MCP 服务器（t_ai_mcp_server）三表；内置 embed 技能手册落库
			// （source=builtin，按 code 查重，重复执行安全）；AI 菜单下插入
			// 「集成」目录 + 「插件管理」页面
			ID: "v1.12.0-ai-plugin-integration",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(
					&aientity.Skill{},
					&aientity.SkillResource{},
					&aientity.McpServer{},
				); err != nil {
					return err
				}

				// 内置技能落库（对齐菜单内置进 SQL 的方式：内容随迁移硬编码，
				// 运行时不依赖 embed，技能统一走 DB 管理）。原生 SQL + NOW() 填充
				// 时间列（model.Model 的时间字段由应用层填充，迁移内裸 Create 会写 NULL）。
				// 按 code 查重，幂等。
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

				// 菜单：AI 目录下插入「集成」目录 + 「插件管理」页面（查重后插入，重复执行安全）
				menus := []struct {
					id     int64
					pid    int64
					uiPath string
					name   string
					code   string
					meta   string
				}{
					{1775967910, 1775967861, "lUgXhO96/V3rTnK8w/", "menu.aiIntegration", "integration", `{"icon":"icon ai/plugin","isKeepAlive":true}`},
					{1775967911, 1775967910, "lUgXhO96/V3rTnK8w/Q9mPy6d2/", "menu.aiPluginManagement", "plugin", `{"icon":"icon ai/plugin","isKeepAlive":true,"routeName":"AiPluginManagement"}`},
				}
				for _, m := range menus {
					var count int64
					if err := tx.Table("t_sys_resource").Where("id = ? AND is_deleted = 0", m.id).Count(&count).Error; err != nil {
						return err
					}
					if count > 0 {
						continue
					}
					if err := tx.Exec("INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (?, ?, ?, 1, 1, ?, ?, ?, ?, 1, 'admin', 1, 'admin', NOW(), NOW(), 0, NULL)",
						m.id, m.pid, m.uiPath, m.name, m.code, m.id, m.meta).Error; err != nil {
						return err
					}
					// 角色权限同步授予（与 v1.12.0-ai-menu-permission 同模式）
					if err := tx.Exec("INSERT INTO t_sys_role_resource (role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (1, ?, 1, 'admin', NOW(), 0, NULL)", m.id).Error; err != nil {
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
