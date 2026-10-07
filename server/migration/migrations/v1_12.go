package migrations

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	aientity "mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/skill"
	alertentity "mayfly-go/internal/alert/domain/entity"
	dbentity "mayfly-go/internal/db/domain/entity"
	fileentity "mayfly-go/internal/file/domain/entity"
	flowentity "mayfly-go/internal/flow/domain/entity"
	labelentity "mayfly-go/internal/label/domain/entity"
	machineentity "mayfly-go/internal/machine/domain/entity"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/jsonx"

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
					matchType dbentity.MaskMatchType
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
		{
			// DB 迁移任务定时列改名 cron_able → cron_enabled，与实体字段名保持一致
			// 列名与字段名一致后，查询条件体可直接传实体/查询dto，无需在仓储层手写列名
			ID: "v1.12.0-db-transfer-cron-column-rename",
			Migrate: func(tx *gorm.DB) error {
				migrator := tx.Migrator()
				if !migrator.HasColumn(&dbentity.DbTransferTask{}, "cron_able") {
					// 旧列不存在：全新库由 AutoMigrate 直接建出 cron_enabled，无需处理
					return nil
				}
				// 两列并存（历史版本曾用别名映射过）：先搬运数据再删旧列，避免改名撞列报错
				if migrator.HasColumn(&dbentity.DbTransferTask{}, "cron_enabled") {
					if err := tx.Exec("UPDATE t_db_transfer_task SET cron_enabled = cron_able WHERE cron_able IS NOT NULL").Error; err != nil {
						return err
					}
					// 不用 Migrator().DropColumn：sqlite 驱动走重建表流程且对该列静默无操作，DROP COLUMN 是 MySQL/SQLite 通用语法
					return tx.Exec("ALTER TABLE t_db_transfer_task DROP COLUMN cron_able").Error
				}
				return migrator.RenameColumn(&dbentity.DbTransferTask{}, "cron_able", "cron_enabled")
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

		// ================================================================
		//  数据同步模块字段补齐（同步任务 + 同步日志新增列）
		// ================================================================

		{
			// 同步任务表新增：同步模式、辅助增量字段、转换规则、过滤条件、空值策略、
			// 软删除配置、Schema 演化、双向同步等字段。AutoMigrate 幂等，仅添加缺失列。
			ID: "v1.12.0-sync-task-columns",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&dbentity.DataSyncTask{})
			},
			Rollback: noopRollback,
		},
		{
			// 同步日志表新增：监控指标字段（耗时、字节数、吞吐量、各操作计数等）。
			ID: "v1.12.0-sync-log-metrics",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&dbentity.DataSyncLog{})
			},
			Rollback: noopRollback,
		},
		{
			// 同步日志表新增：运行日志字段（追加式执行过程记录）。
			ID: "v1.12.0-sync-log-run-log",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&dbentity.DataSyncLog{})
			},
			Rollback: noopRollback,
		},
		{
			// 同步日志表新增 run_id 列：与 taskx.RunGuard 持锁所有权值同步，
			// 供启动收尾/停止检查/水位推进判定“同一次执行”归属。
			ID: "v1.12.0-sync-log-run-id",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&dbentity.DataSyncLog{})
			},
			Rollback: noopRollback,
		},
		{
			// 同步任务表新增 extra 列（model.ExtraData 内嵌字段）：
			// 存放非查询/统计维度的任务配置（游标边界语义、批间 sleep、跳过索引校验等）。
			// 旧的 v1.12.0-sync-task-columns 在本实体尚未内嵌 ExtraData 时已标记为已应用，
			// AutoMigrate 不会重跑，需新开一个登记项触发列的创建。
			ID: "v1.12.0-sync-task-extra-column",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&dbentity.DataSyncTask{})
			},
			Rollback: noopRollback,
		},
		{
			// 迁移任务执行日志表（t_db_transfer_log），对齐数据同步日志架构：
			// 每次执行生成独立记录，支持历史查询与指标统计。
			ID: "v1.12.0-db-transfer-log",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&dbentity.DbTransferLog{})
			},
			Rollback: noopRollback,
		},
		{
			// 迁移任务执行日志新增「执行用途」列（1迁移 2导出文件 3校验）：
			// 区分同一任务下不同性质的执行记录，避免校验记录被误当成迁移。
			// AutoMigrate 幂等，仅补列；历史记录取默认值 1（迁移）。
			ID: "v1.12.0-db-transfer-log-purpose",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&dbentity.DbTransferLog{})
			},
			Rollback: noopRollback,
		},
		{
			ID: "v1.12.0-sync-transfer-permissions",
			Migrate: func(tx *gorm.DB) error {
				resources := []struct {
					id     int64
					pid    int64
					uiPath string
					name   string
					code   string
					weight int
				}{
					{1758412801, 150, "Jra0n7De/Xk3mNpQr/", "menu.dbDataSyncRun", "db:sync:run", 1758412801},
					{1758412802, 150, "Jra0n7De/Yw5tLsVb/", "menu.dbDataSyncStop", "db:sync:stop", 1758412802},
					{1758412803, 1709194669, "SmLcpu6c/Dn8rKwXe/", "menu.dbTransferStop", "db:transfer:stop", 1758412803},
					{1758412804, 1709194669, "SmLcpu6c/Ef2pMzYc/", "menu.dbTransferVerify", "db:transfer:verify", 1758412804},
				}
				for _, r := range resources {
					if err := insertSysResourceWithRole(tx, r.id, r.pid, r.uiPath, 2, r.name, r.code, r.weight, "null"); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: noopRollback,
		},
		{
			// Mongo 数据面权限细化：结构变更（建/删集合与索引）与服务器管理（账号/角色/副本集/参数）
			// 从「文档读写」里独立出来。旧版本只有 mongo:data:save|del，且 run-command 不声明任何权限码，
			// 只读账号即可删库、建管理员账号。
			//
			// 刻意不自动授予公共角色（role_id=1）：这三类都是破坏性或越权能力，
			// 默认仅超级管理员可用（超管按全量资源放行），其他角色需在角色管理里显式勾选。
			ID: "v1.12.0-mongo-structural-permissions",
			Migrate: func(tx *gorm.DB) error {
				for _, r := range []sysResourceDef{
					{1758412901, 1768729911, "ocdrUNaa/GCElqxQr/Kd9mPxQa/", 2, "menu.mongoDdlSave", "mongo:ddl:save", 1758412901, "null"},
					{1758412902, 1768729911, "ocdrUNaa/GCElqxQr/Wn4tRsYb/", 2, "menu.mongoDdlDel", "mongo:ddl:del", 1758412902, "null"},
					{1758412903, 1768729911, "ocdrUNaa/GCElqxQr/Zq7vLmTc/", 2, "menu.mongoCmdAdmin", "mongo:cmd:admin", 1758412903, "null"},
				} {
					if err := insertSysResource(tx, r); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: noopRollback,
		},
		{
			// 导出权限资源单独成一条迁移：它与上面三类同时引入，但必须有自己的 ID。
			// 追加进已执行过的迁移 ID 里，对已升级的库等于没写（gormigrate 按 ID 跳过），
			// 表现是「接口声明了 mongo:data:export 却查不到这条资源」——按钮不显示、接口一律拒绝。
			ID: "v1.12.0-mongo-data-export-permission",
			Migrate: func(tx *gorm.DB) error {
				return insertSysResource(tx, sysResourceDef{1758412904, 1768729911, "ocdrUNaa/GCElqxQr/Hp6yWnBd/", 2, "menu.mongoDataExport", "mongo:data:export", 1758412904, "null"})
			},
			Rollback: noopRollback,
		},
		{
			// MongoConfig 数据面运行配置（执行超时/默认条数/最大结果集/连接池）。
			// 之前这些硬上限散落在 handler 里（如 limit 硬编码 100、context.TODO() 无超时），
			// 部署环境无法调整；收进系统配置后与 DbmsConfig 保持同一形态。
			ID: "v1.12.0-mongo-config",
			Migrate: func(tx *gorm.DB) error {
				var count int64
				if err := tx.Model(&sysentity.Config{}).Where(&sysentity.Config{Key: "MongoConfig"}).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return nil
				}

				// 审计字段必须手动补齐：这里固定了主键 Id（便于幂等与回查），实体钩子会把它当成
				// 「已存在」而跳过 FillBaseInfo，create_time/update_time 为 NOT NULL 会直接报错
				now := time.Now()
				mongoConf := &sysentity.Config{
					Name:       "system.sysconf.mongoConf",
					Key:        "MongoConfig",
					Params:     mongoConfParams,
					Value:      `{"execTl":"60","defaultLimit":"50","maxResultSet":"500","poolSize":"10"}`,
					Remark:     "system.sysconf.mongoConfRemark",
					Permission: "admin,",
				}
				mongoConf.Id = 1775967940
				mongoConf.CreateTime = &now
				mongoConf.CreatorId = 1
				mongoConf.Creator = "admin"
				mongoConf.UpdateTime = &now
				mongoConf.ModifierId = 1
				mongoConf.Modifier = "admin"
				return tx.Create(mongoConf).Error
			},
			Rollback: noopRollback,
		},

		// ================================================================
		//  流程模块
		// ================================================================

		{
			// 机器命令的处置级别改由流程定义触发策略决定，命令配置表不再自带策略字段：
			// 该列长期只被复制到过滤规则里打日志，从未参与任何判定
			ID: "v1.12.0-machine-cmd-conf-drop-stratege",
			Migrate: func(tx *gorm.DB) error {
				if tx.Migrator().HasColumn(&machineentity.MachineCmdConf{}, "stratege") {
					return tx.Migrator().DropColumn(&machineentity.MachineCmdConf{}, "stratege")
				}
				return nil
			},
			Rollback: noopRollback,
		},
		{
			// 触发条件从 Go 模板文本改为结构化触发策略：新增 trigger_policy 列、
			// 为存量流程定义写入内置最佳实践规则包，并删除已废弃的 condition 列
			ID: "v1.12.0-flow-trigger-policy",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&flowentity.Procdef{}); err != nil {
					return err
				}
				if err := fillDefaultTriggerPolicy(tx); err != nil {
					return err
				}
				if tx.Migrator().HasColumn(&flowentity.Procdef{}, "condition") {
					return tx.Migrator().DropColumn(&flowentity.Procdef{}, "condition")
				}
				return nil
			},
			Rollback: noopRollback,
		},
		{
			// 可复用条件组与触发策略变更记录：
			// 前者把常用判断（如「DBA 成员」）沉淀成可引用的一条数据，后者保证
			// 「谁在什么时候撤了哪条规则」可回溯，两张表都是审计与协作的基础设施
			ID: "v1.12.0-flow-rule-segment-policy-history-tables",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(new(flowentity.RuleSegment), new(flowentity.ProcdefPolicyHis))
			},
			Rollback: noopRollback,
		},
		{
			// 条件组标识上的唯一索引必须单独用一条迁移来删。
			// 塞进上面那条建表迁移的 Migrate 里对已升级的库等于没写：gormigrate 按 ID 跳过已执行项，
			// 表现是「删掉一个条件组后，同名标识再也建不出来」——软删除行仍占着唯一键
			ID: "v1.12.0-flow-rule-segment-ref-drop-unique-index",
			Migrate: func(tx *gorm.DB) error {
				migrator := tx.Migrator()
				if !migrator.HasTable(new(flowentity.RuleSegment)) || !migrator.HasIndex(new(flowentity.RuleSegment), "uk_flow_segment_ref") {
					return nil
				}
				return migrator.DropIndex(new(flowentity.RuleSegment), "uk_flow_segment_ref")
			},
			Rollback: noopRollback,
		},
		{
			// 条件组管理菜单 + 按钮权限。
			//
			// 刻意不自动授予公共角色（role_id=1）：条件组被审批策略直接引用，
			// 改一条就等于改一批流程的拦截结果，属于策略级能力，
			// 默认仅超级管理员可用（超管按全量资源放行），其他角色需显式勾选
			ID: "v1.12.0-flow-rule-segment-resources",
			Migrate: func(tx *gorm.DB) error {
				for _, r := range flowRuleSegmentResources {
					if err := insertSysResource(tx, r); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: noopRollback,
		},
		{
			// 连线跳转条件与用户任务完成条件从 Go 模板字符串改为条件树。
			//
			// 条件树存在节点的 extra 里，模板串留在同一位置会让运行期解析直接失败，
			// 因此必须把存量流程图（含运行中实例自带的那份副本）一起翻译过来
			ID: "v1.12.0-flow-condition-tree",
			Migrate: func(tx *gorm.DB) error {
				if err := migrateFlowConditions(tx, "t_flow_procdef"); err != nil {
					return err
				}
				return migrateFlowConditions(tx, "t_flow_procinst")
			},
			Rollback: noopRollback,
		},
		{
			// 主机公钥指纹信任库（TOFU）：首次连接采集指纹备案，指纹失配拒绝连接防中间人。
			// 信任键为原始目标地址 ip:port，不建唯一索引（软删行会撞唯一键，查重由应用层负责）
			ID: "v1.12.0-machine-host-key",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&machineentity.MachineHostKey{})
			},
			Rollback: noopRollback,
		},
		{
			// 机器指标历史采样点（时序，追加写 + 定期物理清理，不做软删除）
			ID: "v1.12.0-machine-metric",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&machineentity.MachineMetric{})
			},
			Rollback: noopRollback,
		},
		{
			// 计划任务增强：超时/重试/结果通知列
			ID: "v1.12.0-machine-cronjob-enhance",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&machineentity.MachineCronJob{})
			},
			Rollback: noopRollback,
		},
	}
}

// flowRuleSegmentResources 可复用条件组的菜单与按钮权限资源
var flowRuleSegmentResources = []sysResourceDef{
	{1748000001, 1708910975, "6egfEVYr/r7LmQpxK/", 1, "menu.flowRuleSegment", "rule-segments", 1748000001, `{"component":"flow/RuleSegmentList","icon":"Share","isKeepAlive":true,"routeName":"RuleSegmentList"}`},
	{1748000002, 1748000001, "6egfEVYr/r7LmQpxK/Sgmt0001/", 2, "menu.flowRuleSegmentSave", "flow:ruleSegment:save", 1748000002, "null"},
	{1748000003, 1748000001, "6egfEVYr/r7LmQpxK/Sgmt0002/", 2, "menu.flowRuleSegmentDel", "flow:ruleSegment:del", 1748000003, "null"},
}

// 旧模板里只有两种完成条件写法（或签、会签），翻译成条件树的等价形态：
// 或签「第一人通过即完成」= 已完成人数 >= 1；会签「所有人都通过」= 完成比例 >= 1。
// 会签改用比例表达是因为条件叶子只支持「字段 与 常量」比较，两个字段之间的关系
// 本来就表达不出来，而换成比例后「过半通过」这类新语义可以直接配
// legacyCondition 构造一条「字段 与 常量比较」的条件节点结构，与实体里的 RuleNode JSON 同形
func legacyCondition(field string, op string, value float64) map[string]any {
	return map[string]any{"kind": "condition", "field": field, "op": op, "value": value}
}

// migrateFlowConditions 把指定表 flow_def 列里的模板条件翻译成条件树。
// 表名固定来自本迁移内部，不接受外部输入，因此拼接表名不构成注入面
func migrateFlowConditions(tx *gorm.DB, table string) error {
	type flowRow struct {
		Id      uint64
		FlowDef string
	}

	// 表不存在时跳过：本迁移只负责翻译内容，建表由各自的建表迁移负责，
	// 顺序或环境差异不该让整个升级流程失败
	if !tx.Migrator().HasTable(table) {
		return nil
	}

	var rows []flowRow
	if err := tx.Table(table).Select("id, flow_def").Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		updated, changed, untranslated, err := translateFlowConditions(row.FlowDef)
		if err != nil {
			logx.Errorf("migrate flow conditions failed, table=%s id=%d: %v", table, row.Id, err)
			continue
		}
		for _, text := range untranslated {
			logx.Errorf("flow condition cannot be translated, table=%s id=%d, it must be reconfigured in the designer: %s", table, row.Id, text)
		}
		if !changed {
			continue
		}
		if err := tx.Table(table).Where("id = ?", row.Id).Update("flow_def", updated).Error; err != nil {
			return err
		}
	}
	return nil
}

// translateFlowConditions 返回翻译后的流程图 JSON、「是否真的改过」与无法翻译的条件原文。
// 解析失败的行原样保留并交给调用方打日志：宁可让这条流程在运行时报「条件无法判定」，
// 也不要静默丢掉一个原本存在的审批分支
func translateFlowConditions(flowDefText string) (string, bool, []string, error) {
	if strings.TrimSpace(flowDefText) == "" {
		return flowDefText, false, nil, nil
	}
	graph := map[string]any{}
	if err := json.Unmarshal([]byte(flowDefText), &graph); err != nil {
		return flowDefText, false, nil, err
	}

	// 无法翻译的条件原文交给调用方打日志：宁可让人去设计器里重新配置，
	// 也不要在升级时猜一个语义出来
	untranslated := make([]string, 0, 2)
	changed := rewriteConditionItems(graph["nodes"], "completionCondition", &untranslated)
	changed = rewriteConditionItems(graph["edges"], "condition", &untranslated) || changed

	if !changed {
		return flowDefText, false, untranslated, nil
	}
	updated, err := json.Marshal(graph)
	if err != nil {
		return flowDefText, false, untranslated, err
	}
	return string(updated), true, untranslated, nil
}

// rewriteConditionItems 把 items 里每个节点/连线 extra 下的模板条件换成条件树
func rewriteConditionItems(items any, conditionKey string, untranslated *[]string) bool {
	list, ok := items.([]any)
	if !ok {
		return false
	}
	changed := false
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		extra, ok := entry["extra"].(map[string]any)
		if !ok {
			continue
		}
		raw, ok := extra[conditionKey].(string)
		if !ok {
			// 已经是条件树（新版设计器写入）或压根没配置，都不需要翻译
			continue
		}
		if strings.TrimSpace(raw) == "" {
			delete(extra, conditionKey)
			changed = true
			continue
		}
		node, translatable := legacyConditionToRuleNode(raw)
		if !translatable {
			// 认不出的写法原样保留：猜一个条件会静默改变审批要求，
			// 而保留原文会让运行期报出「条件无法判定」并点名那条连线/节点
			*untranslated = append(*untranslated, raw)
			continue
		}
		extra[conditionKey] = node
		changed = true
	}
	return changed
}

// legacyConditionToRuleNode 精确匹配旧设计器能产出的两种写法并翻译成等价条件：
// 或签「第一人通过即完成」= 已完成人数 >= 1；会签「所有人都通过」= 完成比例 >= 1。
//
// 只认这两种完整写法而不是做包含匹配：`{{ if gt .nrOfCompleted 2.0 }}` 里同样出现 nrOfCompleted，
// 按包含匹配会被翻译成「>= 1」，那是把「至少 3 人通过」悄悄放宽成「1 人通过」
func legacyConditionToRuleNode(template string) (map[string]any, bool) {
	switch strings.Join(strings.Fields(template), " ") {
	case "{{ eq .nrOfCompleted 1.0 }}":
		return legacyCondition("nrOfCompleted", "gte", 1), true
	case "{{ eq .nrOfAll .nrOfCompleted }}":
		return legacyCondition("nrOfCompletedRate", "gte", 1), true
	default:
		return nil, false
	}
}

// mongoConfParams MongoConfig 配置项定义（v1 JSON schema）
const mongoConfParams = `{"version":1,"cols":2,"fields":[
	{"prop":"execTl","label":"system.sysconf.mongoExecTl","tooltip":"system.sysconf.mongoExecTlPlaceholder"},
	{"prop":"defaultLimit","label":"system.sysconf.mongoDefaultLimit","tooltip":"system.sysconf.mongoDefaultLimitPlaceholder"},
	{"prop":"maxResultSet","label":"system.sysconf.mongoMaxResultSet","tooltip":"system.sysconf.mongoMaxResultSetPlaceholder"},
	{"prop":"poolSize","label":"system.sysconf.mongoPoolSize","tooltip":"system.sysconf.mongoPoolSizePlaceholder"}
]}`

// ================================================================
//  公共辅助函数
// ================================================================

// noopRollback 空回滚（迁移不支持回滚时统一使用）
func noopRollback(tx *gorm.DB) error { return nil }

// fillDefaultTriggerPolicy 为尚未配置触发策略的流程定义写入默认规则包（写操作与危险命令需审批）。
//
// 只处理 trigger_policy 为空的行，保证重复执行安全；已有策略的行不动，避免覆盖管理员已配置的规则
//
// 兜底级别取「不处置」：迁移不该替存量数据做「未配置规则也要审批」这个决定。
//
// 存量流程定义在迁移前根本没有触发判定（旧 condition 为空即放行），下面四条显式规则已把它们原本
// 就在管的写操作/危险命令等价搬过来；再叠一个「需审批」兜底，等于升级当晚把所有绑定资源的未命中
// 操作一并改成要审批。更糟的是没有审批通道的场景（机器命令）：兜底落不了地会被求值期收敛成
// 「禁止执行」（该收敛由 fallback_severity_test 与 machine_trigger_test 锁死，属有意设计），
// 于是管理员把机器加进「生效资源」的那一刻，该机器连 ls 都被硬拒且没有任何出口。
// 想让某个场景受兜底约束，应在策略里显式选「需审批」；界面会用 schema 下发的 strictestSeverity
// 预告该场景实际会落到哪个级别
func fillDefaultTriggerPolicy(tx *gorm.DB) error {
	defaultPolicy := flowentity.NewTriggerPolicy(flowentity.SeverityDisabled)
	defaultPolicy.Checks = []*flowentity.CheckConfig{
		{
			Key: "db.dml-requires-approval", BizType: "db_sql_exec_flow", Severity: flowentity.SeverityRequired,
			Params: map[string]any{"stmtTypes": []string{"insert", "update", "delete", "ddl"}},
		},
		{Key: "db.dml-without-where", BizType: "db_sql_exec_flow", Severity: flowentity.SeverityRequired},
		{Key: "redis.write-cmd", BizType: "redis_run_cmd_flow", Severity: flowentity.SeverityRequired},
		{Key: "redis.dangerous-cmd", BizType: "redis_run_cmd_flow", Severity: flowentity.SeverityRequired},
	}

	return tx.Model(&flowentity.Procdef{}).
		Where("trigger_policy IS NULL OR trigger_policy = ''").
		Update("trigger_policy", jsonx.ToStr(defaultPolicy)).Error
}

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
