package v112

import (
	"strings"
	"time"

	agentopsentity "mayfly-go/internal/agentops/domain/entity"
	aientity "mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/pkg/headless"
	alertentity "mayfly-go/internal/alert/domain/entity"
	"mayfly-go/migration/migrations/migrationx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// V1_12_2 v1.12.2 批次迁移：AgentOps（智能运维）模块与配套的 AI 会话来源隔离、
// 内置 Runbook 指令纠偏。本批新增迁移统一 v1.12.2- 前缀、全部收口在本文件。
//
// 所有迁移均幂等设计：AutoMigrate 按列比对、数据步骤按 key / 列查重后跳过、
// 删列前先回填——重复执行安全（由全链直调重跑与升级路径用例共同覆盖）。
// 原生 SQL 只能用 MySQL 与 SQLite 的方言交集：时间一律 Go 侧传参；
// 裸 ALTER TABLE 的标识符经 clause.Table / clause.Column 交给驱动引用。
func v1122() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			// 统一 Run 聚合建表：cron/event/manual 三类触发的运行生命周期一等公民，
			// 承载状态机、token 成本、会话回放关联与幂等去重（idempotency_key 唯一索引）。
			// AutoMigrate 对全新库建表、对已有库按列比对补齐，天然幂等
			ID: "v1.12.2-agentops-run-table",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&agentopsentity.Run{})
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 定时计划任务建表：声明「何时/何身份/何档/做什么/对哪些资源/超时重试/通知」，
			// 到点由 cron 触发经 RunService.Enqueue 产出 Run（trigger=cron）。AutoMigrate 天然幂等
			ID: "v1.12.2-agentops-task-table",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&agentopsentity.AgentTask{})
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 事件驱动触发器建表：声明「何来源/命中何条件/跑什么(关联任务或内联)/何身份/冷却多久」，
			// 告警或 webhook 事件到达经 TriggerService 匹配 → RunService.Enqueue 产出 Run（trigger=event）。
			// AutoMigrate 对全新库建表、对已有库按列比对补齐，天然幂等
			ID: "v1.12.2-agentops-trigger-table",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&agentopsentity.Trigger{})
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 变更提案建表：只读诊断档经 propose_remediation 工具登记结构化提案（动作 + 锁定参数 +
			// 理由 + 风险级），由 ProposalGateway 发起 flow 审批工单；审批通过回调 → 执行档受控执行 →
			// 只读校验档复查 → 结论回写。是「诊断→提案→审批→受控执行→验证」闭环的事实源。
			// AutoMigrate 对全新库建表、对已有库按列比对补齐，天然幂等
			ID: "v1.12.2-agentops-proposal-table",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&agentopsentity.Proposal{})
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// AgentOps（智能运维）菜单 + 按钮权限资源，挂在 AI 父菜单下成「智能运维」分组。
			//
			// 刻意不自动授予公共角色（role_id=1）：定时任务会以「负责人账号」身份触发无头 Agent
			// 运行（消耗 LLM 配额、按 owner 收敛工具权限、可手动触发/取消），属于高权限运维能力，
			// 默认仅超级管理员可见（超管按全量资源放行），其他角色需显式勾选授权（对齐条件组菜单）
			ID: "v1.12.2-agentops-menu-resources",
			Migrate: func(tx *gorm.DB) error {
				for _, r := range agentopsResources {
					if err := migrationx.InsertResource(tx, r); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 内置 Runbook 任务模板：开箱即用的只读巡检/日报模板（DB 慢查询巡检、机器磁盘负载巡检、
			// 每日运维日报），Instruction 经 $skill-code 绑定既有内置技能 db-ops-guide/machine-ops-guide。
			// 默认禁用落库（不绑资源、admin 为默认负责人），用户 review + 绑定资源/负责人/通知后启用。
			// 依赖 v1.12.2-agentops-task-table 建表在先；seedBuiltinRunbooks 按 key 查重，幂等
			ID: "v1.12.2-agentops-runbook-templates",
			Migrate: func(tx *gorm.DB) error {
				return seedBuiltinRunbooks(tx)
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// AgentOps 阶段4 补全列：Run/Task 新增 token 预算熔断（token_budget）、影子模式（dry_run），
			// Run 另加人类反馈（feedback/feedback_note，学习闭环）。AutoMigrate 按列比对补齐，天然幂等
			ID: "v1.12.2-agentops-run-task-enhance-columns",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&agentopsentity.Run{}); err != nil {
					return err
				}
				return tx.AutoMigrate(&agentopsentity.AgentTask{})
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// AgentOps 阶段4 按钮权限资源（挂在执行历史页下）：多 Agent 编排发起 / 运行反馈 / 离线评测。
			// 同样不自动授予公共角色（与 v1.12.2-agentops-menu-resources 同一授权口径）
			ID: "v1.12.2-agentops-menu-resources-extend",
			Migrate: func(tx *gorm.DB) error {
				for _, r := range agentopsExtendResources {
					if err := migrationx.InsertResource(tx, r); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// Conversation 补 source 列（user/system）：系统任务触发的 headless 会话
			// 不进用户 AI 助手会话列表，与用户发起的 chat 区分（回放仍按 convId 直达）。
			// 存量回填双口径：① 凡被 agentops 运行引用（conversation_id 关联）的会话；
			// ② 评测/编排会话标题前缀（评审运行的 conversationId 未必回写 run 行）。
			// 顺带把 t_ai_turn_item.extra 扩为 mediumtext（varchar(2000) 曾被参数补全
			// 中断事件携带的完整选项列表撑爆，Error 1406 致整批 turn items 落库失败；
			// 仅 mysql 需要 ALTER，sqlite 动态类型无长度限制）
			ID: "v1.12.2-ai-conversation-source",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&aientity.Conversation{}, &aientity.TurnItem{}); err != nil {
					return err
				}
				if tx.Dialector.Name() == "mysql" {
					if err := tx.Exec("ALTER TABLE t_ai_turn_item MODIFY COLUMN `extra` mediumtext").Error; err != nil {
						return err
					}
				}
				return tx.Exec("UPDATE t_ai_conversation SET source = 'system' WHERE " +
					"id IN (SELECT conversation_id FROM t_aiops_run WHERE conversation_id IS NOT NULL AND conversation_id > 0) " +
					"OR title LIKE '[评测%' OR title LIKE '[编排%'").Error
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 内置 Runbook 指令纠偏：补「资源发现 + 库名/schema 约定」公共引导。
			// 曾致日报任务失败：无目标资源时模型凭空猜资源 id/把资源显示名当 schema 名，
			// 工具报错中断整轮运行。按 key 覆盖 instruction（仅指令列，不碰其他配置）
			ID: "v1.12.2-agentops-runbook-instruction-guide",
			Migrate: func(tx *gorm.DB) error {
				return updateBuiltinRunbookInstructions(tx)
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 运行行新增两列，必须单独一个迁移步，不能只改实体：
			// 前面的建表/补列 AutoMigrate 步在已跑过它们的库上不会重跑（迁移按 ID 记账只执行一次），
			// 实体新加的列因此永远补不上——实测升级库上报 Unknown column 'extra'，所有入队直接失败。
			//   runner_instance：启动清扫按归属收敛，不跨实例误杀在途运行
			//   extra：一次性工具许可（受控执行只能调已批准那个工具）随运行自持
			ID: "v1.12.2-agentops-run-instance-and-extra-columns",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&agentopsentity.Run{})
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 编排子运行的归属列（parent_run_id）：同样必须单独一步，不能靠已录库不会重跑的旧 AutoMigrate 步补齐。
			// 配合新触发类型 orchestration，执行历史可以「触发方式=编排」筛出整棵编排树并汇总成本
			ID: "v1.12.2-agentops-run-parent-id-column",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&agentopsentity.Run{})
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 展示型列收敛进 model.Extra（docs/server/domain.md：关联名称/备注类展示字段不占独立列）：
			// 实体上的 owner_username / remark / feedback_note 已改为瞬态字段 + Extra 落库（由
			// BeforeSave/AfterFind 钩子双向同步）。本步先 AutoMigrate 补出各表 extra 列，再把旧列
			// 已有值回填进 extra 并删列——顺序不可颠倒，删列后没有值可回填（历史行负责人/备注会凭空变空）
			ID: "v1.12.2-agentops-display-columns-into-extra",
			Migrate: func(tx *gorm.DB) error {
				for _, dest := range []any{
					&agentopsentity.Run{}, &agentopsentity.AgentTask{},
					&agentopsentity.Trigger{}, &agentopsentity.Proposal{},
				} {
					if err := tx.AutoMigrate(dest); err != nil {
						return err
					}
				}
				return collapseAgentOpsDisplayColumns(tx)
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 目标资源列收敛进指令引用：AI 助手输入框已支持 @资源 引用（芯片段自带
			// machineId/dbName 等定位信息），再留一列手写 JSON 等于同一件事两个入口。
			// 先把存量值折进 instruction 再删列——只删列会让已有任务下次运行时没有目标（无报错的行为改变）
			ID: "v1.12.2-agentops-target-resources-into-instruction",
			Migrate: func(tx *gorm.DB) error {
				return foldAgentOpsTargetResourcesIntoInstruction(tx)
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 补齐「查看全部运行」权限点资源（run 侧可见面收窄 canQueryAllRuns 依赖它）。
			//
			// 必须独立成条而不是并入 agentopsExtendResources：那个列表所属的迁移 ID 已在库中
			// migrations 表记账，gormigrate 按 ID 跳过已完成项，往已应用的迁移函数体里追加数据
			// 对升级过的库等于没写（代码里看着登记了，库里永远查不到）。InsertResource 自身幂等，
			// 负库（该资源已由上一轮写入）与新库都安全
			ID: "v1.12.2-agentops-run-query-all-resource",
			Migrate: func(tx *gorm.DB) error {
				return migrationx.InsertResource(tx, agentopsRunQueryAllResource)
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 为「AI 变更提案审批」落一条最小可用的单级审批流程定义。
			//
			// 为什么必须 seed 而不是要文档求管理员手建：提案登记时按 defKey 找流程定义，找不到走
			// fail-soft（提案停在 pending 并提示联系管理员配置），新装库里这条链开箱即断，
			// 而治理页只会把 pending 数量涨上去、没有任何引导信号
			ID: "v1.12.2-agentops-remediation-procdef-seed",
			Migrate: func(tx *gorm.DB) error {
				return seedAgentRemediationProcdef(tx)
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// seed「运维诊断结果通知」模板：agentops 触发器/任务结果通知的参数是诊断专用的，
			// 新装库只有 flow 的工单审批模板，选它渲染全是 <no value>（真机实测），故补一条语义匹配、
			// 与 notifyResult 参数同源的默认模板。只给内容不绑渠道（渠道环境特定，由管理员配置后关联）
			ID: "v1.12.2-agentops-diagnosis-notify-tmpl",
			Migrate: func(tx *gorm.DB) error {
				return seedAgentDiagnosisNotifyTmpl(tx)
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 把存量运行的「形态」从标题中文前缀搬进 Extra（runRole）。
			//
			// 为什么必须新开一条而不是改已应用的迁移：gormigrate 按 ID 跳过已完成项，往旧迁移函数体
			// 里追加逻辑对升级库静默不生效（本仓已踩过多次）。新库跑到这一步时 t_aiops_run 为空，
			// 本步自然空转，写入侧已直接带 runRole。
			//
			// 不搬的后果：历史行没有 runRole，前端要么给它们再造一份「标题前缀→标签」镜像表
			// （等于把跨语言双真源问题搬进前端），要么老运行的形态标签永远缺失。搬一次就彻底断开
			// 「按标题分类」这条路径
			ID: "v1.12.2-agentops-run-role-from-title",
			Migrate: func(tx *gorm.DB) error {
				return backfillAgentOpsRunRoleFromTitle(tx)
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 告警通知日志补策略来源类型列（policy_type），必须单独一个迁移步，不能只改实体：
			// 建表步 v1.12.0-alert-tables 在已跑过它的库上不会重跑（迁移按 ID 记账只执行一次），
			// 实体新加的列因此永远补不上；而写入侧是裸 Create（全列 INSERT），升级库上会直接报
			// Unknown column 'policy_type'，整批通知日志丢失（与 agentops-run-extra-columns 同一成因）。
			//
			// 不补的后果（真机实测）：policy_id 一列同时承载 t_alert_notify_policy.id 与
			// t_alert_escalation.id，两表主键空间独立且已重叠——日志里的 policy_id=11 在通知策略表
			// 查无此行，实为升级策略 11，「这条通知为什么发给了这个渠道」无从溯源。
			// 存量行无法回溯来源，policy_type 留 0（无归因），不猜也不回填
			ID: "v1.12.2-alert-notify-log-policy-type-column",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&alertentity.AlertNotifyLog{})
			},
			Rollback: migrationx.NoopRollback,
		},

		{
			// 给缺少基础权限码的菜单补齐 xxx:base 权限码。
			//
			// 批次号跟随应用版本（internal/pkg/config/app.go 的 Version 仍为 v1.12.2），
			// 因此新迁移收口在本批次文件、沿用 v1.12.2- 前缀，不另建 v1.12.3 / v1.13 批次。
			//
			// 背景：「我的资源 / 我的流程 / 我的任务」三个菜单此前只有菜单行，没有任何按钮权限码。
			// 后端下发的 permissions 数组只含权限码、不含菜单 code，于是首页等入口无法仅凭权限码
			// 判断「该不该显示跳转入口」。这里按「一个可进入的页面配一个基础码」的既有约定补齐
			//（同 db:transfer、ai:chat），并把新码授予所有已拥有对应菜单的角色：
			// 只授 COMMON 角色会让自定义角色升级后入口消失
			ID: "v1.12.2-menu-base-permission-codes",
			Migrate: func(tx *gorm.DB) error {
				defs := []migrationx.ResourceDef{
					// 我的资源（菜单 id 1756122788）
					migrationx.Resource(1777000001, 1756122788, "ocdrUNaa/RsxBsc01/", 2, "menu.myResourceBase", "resource:base", 1777000001, "null"),
					// 工单流程 / 我的流程（菜单 id 1709103180）
					migrationx.Resource(1777000002, 1709103180, "6egfEVYr/oNCIbynR/FlxBsc01/", 2, "menu.myFlowBase", "flow:base", 1777000002, "null"),
					// 我的待办（菜单 id 1709045735）
					migrationx.Resource(1777000003, 1709045735, "6egfEVYr/3r3hHEub/FlxBsc02/", 2, "menu.myTaskBase", "flow:task:base", 1777000003, "null"),
				}

				for _, def := range defs {
					if err := migrationx.InsertResource(tx, def); err != nil {
						return err
					}
					if err := migrationx.GrantResourceToParentRoles(tx, def.Pid, def.Id); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: migrationx.NoopRollback,
		},
	}
}

// backfillAgentOpsRunRoleFromTitle 按历史标题前缀识别运行形态，写进 extra.runRole 并把前缀从标题剥掉。
//
// 逐行读改写而不用 JSON 函数：跨方言（mysql/sqlite/pg）的 JSON 修改函数形态不一致，而带前缀的
// 行体量小（只来自编排/评测/受控执行三条链），Go 侧处理最稳。空 Extra 用 COALESCE 兼容三方言
// （IFNULL 在 pg 上不存在）。`[` 在 LIKE 里不是通配符，无需 ESCAPE（加了反而在部分方言上报错）。
// extra 是脏 JSON 时跳过该行不覆盖：宁可少一个标签，也不能抹掉该行已有的许可/工具失败记录。
func backfillAgentOpsRunRoleFromTitle(tx *gorm.DB) error {
	type row struct {
		Id    uint64 `gorm:"column:id"`
		Title string `gorm:"column:title"`
		Extra string `gorm:"column:extra"`
	}
	var rows []*row
	// LIKE 在这里只是「定位待修存量」的一次性扫描条件，不是运行期分类判据（分类唯一真源是 runRole）
	if err := tx.Table("t_aiops_run").
		Select("id, title, COALESCE(extra, '') AS extra").
		Where("title LIKE '[%'").Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		role, bare := agentopsentity.DetectLegacyRunRoleTitle(r.Title)
		if role == "" {
			continue // 不是本模块的形态前缀（如用户自建标题恰好以 [ 开头）：不动它
		}
		extra := collx.M{}
		if strings.TrimSpace(r.Extra) != "" {
			m, merr := jsonx.ToMapByStr(r.Extra)
			if merr != nil || m == nil {
				continue
			}
			extra = m
		}
		extra[agentopsentity.ExtraKeyRunRole] = role
		if err := tx.Table("t_aiops_run").Where("id = ?", r.Id).Updates(map[string]any{
			"title": bare,
			"extra": jsonx.ToStr(extra),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// agentopsResources 智能运维（AgentOps）的菜单与按钮权限资源。
//
// 挂在 AI 父菜单（1775967861，code=/ai）下，与 AI 助手/集成/设置并列成一个「智能运维」分组：
// 分组节点无 routeName（侧栏展开为父级，落地页由前端归一化补 redirect 到首个可渲染子菜单），
// 两个叶子页（任务管理 / 执行历史）的 routeName 对应 frontend/src/views/agentops/route.ts 的组件键。
// 按钮资源（typ=2）的 code 即权限码，与后端 RequiredPermissionCode、前端 v-auth 同源对齐。
var agentopsResources = []migrationx.ResourceDef{
	// 分组：智能运维
	migrationx.Resource(1775968000, 1775967861, "lUgXhO96/Ai0psGrp/", 1, "menu.aiops", "agentops", 1775968000, `{"icon":"Cpu","isKeepAlive":true}`),

	// 任务管理页 + 按钮权限（保存/删除/手动触发）
	migrationx.Resource(1775968001, 1775968000, "lUgXhO96/Ai0psGrp/Ai0psTsk/", 1, "menu.aiopsTask", "task", 1775968001, `{"component":"agentops/TaskList","icon":"Timer","isKeepAlive":true,"routeName":"AgentopsTaskList"}`),
	migrationx.Resource(1775968002, 1775968001, "lUgXhO96/Ai0psGrp/Ai0psTsk/Ai0psTs1/", 2, "menu.aiopsTaskSave", "agentops:task:save", 1775968002, "null"),
	migrationx.Resource(1775968003, 1775968001, "lUgXhO96/Ai0psGrp/Ai0psTsk/Ai0psTs2/", 2, "menu.aiopsTaskDel", "agentops:task:del", 1775968003, "null"),
	migrationx.Resource(1775968004, 1775968001, "lUgXhO96/Ai0psGrp/Ai0psTsk/Ai0psTs3/", 2, "menu.aiopsTaskRun", "agentops:task:run", 1775968004, "null"),

	// 执行历史页 + 按钮权限（取消运行）
	migrationx.Resource(1775968010, 1775968000, "lUgXhO96/Ai0psGrp/Ai0psRun/", 1, "menu.aiopsRun", "run", 1775968010, `{"component":"agentops/RunList","icon":"Document","isKeepAlive":true,"routeName":"AgentopsRunList"}`),
	migrationx.Resource(1775968011, 1775968010, "lUgXhO96/Ai0psGrp/Ai0psRun/Ai0psRn1/", 2, "menu.aiopsRunCancel", "agentops:run:cancel", 1775968011, "null"),

	// 触发器管理页 + 按钮权限（保存/删除）：事件驱动触发（告警联动/webhook）的配置入口
	migrationx.Resource(1775968020, 1775968000, "lUgXhO96/Ai0psGrp/Ai0psTrg/", 1, "menu.aiopsTrigger", "trigger", 1775968020, `{"component":"agentops/TriggerList","icon":"Bell","isKeepAlive":true,"routeName":"AgentopsTriggerList"}`),
	migrationx.Resource(1775968021, 1775968020, "lUgXhO96/Ai0psGrp/Ai0psTrg/Ai0psTg1/", 2, "menu.aiopsTriggerSave", "agentops:trigger:save", 1775968021, "null"),
	migrationx.Resource(1775968022, 1775968020, "lUgXhO96/Ai0psGrp/Ai0psTrg/Ai0psTg2/", 2, "menu.aiopsTriggerDel", "agentops:trigger:del", 1775968022, "null"),

	// 变更提案页 + 按钮权限（删除）：自主运维闭环的「提案→审批→受控执行→验证」记录入口。
	// 提案由 Agent 只读诊断时经 propose_remediation 工具登记（非用户手建），故仅「查看/删除终态」，
	// 审批动作在 flow 工单页完成，此处不设编辑权限
	migrationx.Resource(1775968030, 1775968000, "lUgXhO96/Ai0psGrp/Ai0psPps/", 1, "menu.aiopsProposal", "proposal", 1775968030, `{"component":"agentops/ProposalList","icon":"Stamp","isKeepAlive":true,"routeName":"AgentopsProposalList"}`),
	migrationx.Resource(1775968031, 1775968030, "lUgXhO96/Ai0psGrp/Ai0psPps/Ai0psPp1/", 2, "menu.aiopsProposalDel", "agentops:proposal:del", 1775968031, "null"),

	// 治理看板页 + 按钮权限（急停）：全局急停开关 + 运行 KPI/成本统计 + 资源盘点 + 一键清场。
	// 急停是全局性危险动作，独立权限点；统计读取随菜单放行
	migrationx.Resource(1775968040, 1775968000, "lUgXhO96/Ai0psGrp/Ai0psGov/", 1, "menu.aiopsGovernance", "governance", 1775968040, `{"component":"agentops/Governance","icon":"Odometer","isKeepAlive":true,"routeName":"AgentopsGovernance"}`),
	migrationx.Resource(1775968041, 1775968040, "lUgXhO96/Ai0psGrp/Ai0psGov/Ai0psGo1/", 2, "menu.aiopsGovernanceKill", "agentops:governance:killswitch", 1775968041, "null"),
	// 「以他人账号身份派发运行」权限点（任务/触发器/编排三处共用）：运行身份就是 Agent 可调工具与
	// 可见资源的权限边界，故不并入保存权限。不勾也能保存（owner 强制为本人，谁建谁负责），
	// 勾上后才能把 owner 指定成别的账号
	migrationx.Resource(1775968042, 1775968040, "lUgXhO96/Ai0psGrp/Ai0psGov/Ai0psGo2/", 2, "menu.aiopsGovernanceImpersonate", "agentops:owner:impersonate", 1775968042, "null"),
}

// agentopsExtendResources AgentOps 阶段4 追加的按钮权限资源（挂在执行历史页 1775968010 下）：
// 编排发起（多 Agent 编排入口）、运行反馈（点赞/点踩回流记忆）、离线评测（历史事件重放打分）。
// 「查看全部运行」不在此列——它所属的迁移 ID 已应用，只能另开一条，见 agentopsRunQueryAllResource
var agentopsExtendResources = []migrationx.ResourceDef{
	migrationx.Resource(1775968012, 1775968010, "lUgXhO96/Ai0psGrp/Ai0psRun/Ai0psRn2/", 2, "menu.aiopsOrchestrationRun", "agentops:orchestration:run", 1775968012, "null"),
	migrationx.Resource(1775968013, 1775968010, "lUgXhO96/Ai0psGrp/Ai0psRun/Ai0psRn3/", 2, "menu.aiopsRunFeedback", "agentops:run:feedback", 1775968013, "null"),
	migrationx.Resource(1775968014, 1775968010, "lUgXhO96/Ai0psGrp/Ai0psRun/Ai0psRn4/", 2, "menu.aiopsRunEval", "agentops:run:eval", 1775968014, "null"),
}

// seedAgentRemediationProcdef 落一条最小可用的「AI 变更提案审批」流程定义（def_key 与
// agentops/application 里的 defaultRemediationProcDefKey 一致）。
//
// 幂等且**不覆盖**：def_key 已存在（含存量环境已手工配过）就直接跳过——这里是
// 「新库开箱可用」的兜底，不能把管理员调好的审批人/会签/风险分级重置成单级默认值。
//
// 默认审批人给超管（账号 id=1）：提案本身就是高危变更，默认值必须是「能马上跑通」的那个；
// 备注里明写可在流程管理中改成会签或按风险分级，避免被当成唯一形态
func seedAgentRemediationProcdef(tx *gorm.DB) error {
	const defKey = "agent_remediation"

	var existing int64
	if err := tx.Table("t_flow_procdef").
		Where("def_key = ? AND is_deleted = 0", defKey).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	const (
		startKey = "seed-start"
		taskKey  = "seed-usertask"
		endKey   = "seed-end"
	)
	flowDef := map[string]any{
		"nodes": []any{
			map[string]any{"name": "开始", "key": startKey, "type": "start"},
			map[string]any{
				"name": "提案审批", "key": taskKey, "type": "usertask",
				"extra": map[string]any{
					// 候选人写账号 id 字符串（与手工配置的同形）；单人即完成，所以会签比例阀值给 1
					"candidates": []string{"1"},
					"completionCondition": map[string]any{
						"kind": "condition", "field": "nrOfCompleted", "op": "gte", "value": 1,
					},
				},
			},
			map[string]any{"name": "结束", "key": endKey, "type": "end"},
		},
		"edges": []any{
			map[string]any{"name": "", "key": "seed-e1", "sourceNodeKey": startKey, "targetNodeKey": taskKey},
			map[string]any{"name": "", "key": "seed-e2", "sourceNodeKey": taskKey, "targetNodeKey": endKey},
		},
	}

	now := time.Now()
	return tx.Table("t_flow_procdef").Create(map[string]any{
		"name":           "AI变更提案审批",
		"def_key":        defKey,
		"flow_def":       jsonx.ToStr(flowDef),
		"status":         1,
		"trigger_policy": "null",
		"remark":         "迁移自动创建的最小单级审批流（审批人=超级管理员）；可在「流程管理」里改审批人、加会签或按风险分级",
		"is_deleted":     0,
		"creator":        "system",
		"creator_id":     0,
		"modifier":       "system",
		"modifier_id":    0,
		"create_time":    now,
		"update_time":    now,
	}).Error
}

// agentopsRunQueryAllResource 「查看全部运行」权限点资源（执行历史列表 GET 按平台惯例不挂权限码，
// 可见面改由 owner 收窄，本码是放开全量的开关）。单独声明是因为它不能再追加到上方那个已应用迁移
// 使用的列表里（见 v1122 中 v1.12.2-agentops-run-query-all-resource 的注释）
var agentopsRunQueryAllResource = migrationx.Resource(
	1775968015, 1775968010, "lUgXhO96/Ai0psGrp/Ai0psRun/Ai0psRn5/", 2,
	"menu.aiopsRunQueryAll", "agentops:run:queryAll", 1775968015, "null",
)

// 内置 Runbook 任务模板（对齐内置技能落库方式：内容随迁移硬编码落库 t_aiops_task，运行时不依赖 embed）。
//
// Runbook 即技能：模板 Instruction 通过 $skill-code 显式引用既有内置技能
// （db-ops-guide / machine-ops-guide），运行时由技能目录渐进式披露 + skill_read 按需加载全文；
// 模板本身只承载「巡检目标 / 步骤 / 输出格式 / 安全约束」的编排知识，与技能手册解耦。
//
// 默认「禁用」落库：模板不绑定具体资源、以 admin 为默认负责人，需用户 review、绑定目标资源
// 与负责人、按需调整 cron / 通知后再启用——避免开箱即以空资源自动跑、消耗 LLM 配额（对齐只读安全基线）。
type builtinRunbookTemplate struct {
	Key           string   // 任务唯一 key（幂等去重基准；创建后不可改）
	Name          string   // 任务名称（展示）
	Instruction   []string // 指令正文行（含反引号故逐行存储，避免 raw string 限制）
	Cron          string   // cron 表达式（5/6 字段皆可，见 pkg/scheduler 解析器）
	ProfileId     string   // Agent 装配档（只读诊断档 diagnostician）
	NotifyType    int8     // 结果通知方式（见 agentopsentity.TaskNotify*）
	MisfirePolicy int8     // 错过触发策略（见 agentopsentity.MisfirePolicy*）
	RetryTimes    int8     // 失败重试次数（仅超时/连接类）
	Remark        string   // 备注（引导用户如何启用）
}

// runbookGuideLines 内置 Runbook 公共执行引导（置于各模板指令开头）：统一资源发现与
// 命名约定，规避模型凭空猜资源 id / 把资源显示名当 schema 名导致的工具报错与 token 浪费
// （曾致日报任务失败：QueryMetrics resource not found、DbQueryData Unknown database「本地库」）
var runbookGuideLines = []string{
	"资源发现：若「目标资源」为空，先调用 ListResources 盘点可用数据库与机器，再对每个目标执行；工具入参中的资源 id 一律取自工具返回结果，禁止凭空猜测。",
	"库名/schema 约定：数据库工具的 dbName 是实例内的实际库名（schema），不是资源显示名（如「本地库」）；先对目标实例执行 SHOW DATABASES 获取真实库名，再用于 SQL 的 table_schema 等条件。",
	"参数完备约定：数据库查询务必同时携带 dbId 与 dbName（均取自 ListResources 返回或目标资源上下文），避免触发参数补全交互中断本轮无人值守运行。",
	"",
}

// fullInstruction 模板完整指令 = 公共引导 + 模板正文（seed 与既有任务纠偏更新共用）
func fullInstruction(t builtinRunbookTemplate) string {
	lines := append(append([]string{}, runbookGuideLines...), t.Instruction...)
	return strings.Join(lines, "\n")
}

var BuiltinRunbookTemplates = []builtinRunbookTemplate{
	{
		Key:           "runbook_db_slowlog",
		Name:          "DB 慢查询巡检",
		ProfileId:     headless.ProfileDiagnostician,
		Cron:          "0 2 * * *", // 每天 02:00
		NotifyType:    agentopsentity.TaskNotifyNone,
		MisfirePolicy: agentopsentity.MisfirePolicySkip,
		RetryTimes:    1,
		Remark:        "内置 Runbook 模板（默认禁用）：绑定目标数据库、负责人与通知模板后启用；全程只读巡检，任何变更请走提案审批。",
		Instruction: []string{
			"你是数据库运维诊断助手。请对「目标资源」中指定的数据库执行一次只读慢查询巡检。全程只读，不做任何变更（如确需变更，请在结论中标注“建议走变更提案审批”）。",
			"",
			"请先用 skill_read 加载并遵循技能 $db-ops-guide（数据库操作手册：只读优先、分页查询、SQL 规范与危险操作防护）。",
			"",
			"巡检步骤：",
			"1. 确认目标实例可连通，读取其类型与版本。",
			"2. 采集慢查询：MySQL 查 sys.statement_analysis / performance_schema 或 mysql.slow_log，PostgreSQL 查 pg_stat_statements，按平均或总耗时取 Top 10；其他类型优先用只读系统视图，避免全表扫描。",
			"3. 逐条分析 Top 慢查询：SQL 摘要、执行次数、平均/总耗时、疑似原因（缺索引/全表扫描/大结果集/锁等待）、优化建议（仅建议，不执行任何 DDL/DML）。",
			"4. 只读检查是否存在长时间未结束的事务或锁等待。",
			"",
			"输出结构化报告：",
			"- 概览：实例、采集时间、慢查询条数、健康评级（正常/关注/风险）。",
			"- Top 慢查询明细表。",
			"- 优化建议清单（按收益排序）。",
			"- 风险与后续动作（需人工介入的变更明确标注“建议走变更提案审批”）。",
			"",
			"安全约束：所有查询必须带 LIMIT/分页；禁止任何写操作（INSERT/UPDATE/DELETE 及 DDL）；数据量大时只报告摘要与统计，不倾倒原始数据。",
		},
	},
	{
		Key:           "runbook_machine_resource",
		Name:          "机器磁盘/负载巡检",
		ProfileId:     headless.ProfileDiagnostician,
		Cron:          "0 */6 * * *", // 每 6 小时
		NotifyType:    agentopsentity.TaskNotifyNone,
		MisfirePolicy: agentopsentity.MisfirePolicySkip,
		RetryTimes:    1,
		Remark:        "内置 Runbook 模板（默认禁用）：绑定目标机器、负责人与通知模板后启用；全程只读探查，任何处置请走提案审批。",
		Instruction: []string{
			"你是机器运维诊断助手。请对「目标资源」中指定的机器执行一次只读健康巡检（磁盘、负载、内存、关键进程）。全程只读，不做任何变更（如确需处置，请在结论中标注“建议走变更提案审批”）。",
			"",
			"请先用 skill_read 加载并遵循技能 $machine-ops-guide（机器操作手册：只读探查优先、单步执行、危险命令防护）。",
			"",
			"巡检步骤：",
			"1. 磁盘：`df -h` 查看各分区使用率，标记使用率 > 80% 的分区；对高占用分区用 `du -sh <dir>` 定位大目录（只读）。",
			"2. 负载：`uptime` 或 `cat /proc/loadavg` 读取 1/5/15 分钟负载，结合 CPU 核数（`nproc`）判断是否过载。",
			"3. 内存：`free -h` 查看内存与 swap，标记 swap 高占用或可用内存不足。",
			"4. 进程：`ps aux` 查看 CPU/内存 Top 进程，识别异常占用。",
			"5. 端口（可选）：`ss -tlnp` 或 `netstat -tlnp` 查看监听端口是否符合预期。",
			"",
			"输出结构化报告：",
			"- 概览：主机、采集时间、整体健康评级（正常/关注/风险）。",
			"- 分项结论：磁盘 / 负载 / 内存 / 进程，每项给出关键指标与是否越阈值。",
			"- 风险清单与处理建议（仅建议，不执行 rm/kill/重启等危险命令；需处置的明确标注“建议走变更提案审批”）。",
			"",
			"安全约束：仅使用只读探查命令；单步执行、每步确认；禁止 rm/mv/kill/chmod 及服务启停等危险命令；输出做摘要归纳，不原样倾倒日志。",
		},
	},
	{
		Key:           "runbook_daily_ops_report",
		Name:          "每日运维日报",
		ProfileId:     headless.ProfileDiagnostician,
		Cron:          "0 8 * * *", // 每天 08:00
		NotifyType:    agentopsentity.TaskNotifyNone,
		MisfirePolicy: agentopsentity.MisfirePolicyCatchUpOnce, // 重启错过窗口仍补发一份日报
		RetryTimes:    1,
		Remark:        "内置 Runbook 模板（默认禁用）：绑定目标库/机与负责人，并配置通知模板、改为“总是通知”后启用，即可每日推送运维日报。",
		Instruction: []string{
			"你是运维日报助手。请汇总生成一份「每日运维日报」，覆盖「目标资源」中指定的数据库与机器。全程只读，不做任何变更（如确需处置，请标注“建议走变更提案审批”）。",
			"",
			"请先用 skill_read 加载并遵循技能 $db-ops-guide（数据库操作手册）与 $machine-ops-guide（机器操作手册）。",
			"",
			"日报内容：",
			"1. 数据库：对每个目标库汇总当日健康度——慢查询数量与 Top 3、连接数、是否有长事务/锁等待、容量增长趋势（只读查询系统视图）。",
			"2. 机器：对每台目标机汇总——磁盘使用率（标记 > 80% 分区）、负载、内存/swap、CPU Top 进程（只读命令）。",
			"3. 异常与风险：列出所有越阈值项与疑似风险，按严重程度排序。",
			"4. 结论与建议：给出当日整体评级（正常/关注/风险）与次日关注点；需人工处置的变更明确标注“建议走变更提案审批”。",
			"",
			"输出要求：Markdown 结构化日报，含“概览 / 数据库 / 机器 / 风险 / 建议”分节；数据以摘要与统计为主，不倾倒原始明细。",
			"",
			"安全约束：所有数据库查询带 LIMIT/分页；机器仅用只读命令；禁止任何写操作与危险命令。",
		},
	},
}

// seedBuiltinRunbooks 内置 Runbook 任务模板落库（按 key 查重，幂等）。
//
// t_aiops_task 的 key 上有唯一索引 uk_aiops_task_key，故查重刻意不加 is_deleted 过滤：
// 无论模板行是否被软删，只要 key 已存在即跳过，避免唯一索引冲突（迁移本就仅执行一次，
// 查重是防重复执行/测试的兜底）。owner 缺省为 admin（id=1），status 一律禁用，待用户启用。
// 注意：key 是 MySQL 保留字，裸 SQL 中必须反引号转义（sqlite 无此要求，勿删）
//
// 负责人账号名与备注不写独立列（已收敛进 extra，见 entity/extra.go）：建表时这两列根本不存在，
// 旧写法在此会直接报错。键名引用 domain 常量，与实体 AfterFind 读回的键同源，不另立字面量。
func seedBuiltinRunbooks(tx *gorm.DB) error {
	for _, t := range BuiltinRunbookTemplates {
		var count int64
		if err := tx.Model(&agentopsentity.AgentTask{}).Where("`key` = ?", t.Key).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		now := time.Now()
		extra := collx.M{agentopsentity.ExtraKeyOwnerUsername: "admin"}
		if t.Remark != "" {
			extra[agentopsentity.ExtraKeyRemark] = t.Remark
		}
		if err := tx.Exec(
			"INSERT INTO t_aiops_task (name, `key`, instruction, cron, profile_id, "+
				"owner_account_id, extra, status, timeout_seconds, retry_times, max_step, "+
				"notify_type, notify_tmpl_code, misfire_policy, last_fire_time, "+
				"creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) "+
				"VALUES (?, ?, ?, ?, ?, 1, ?, ?, 0, ?, 0, ?, '', ?, NULL, 1, 'admin', 1, 'admin', ?, ?, 0, NULL)",
			t.Name, t.Key, fullInstruction(t), t.Cron, t.ProfileId, jsonx.ToStr(extra),
			agentopsentity.AgentTaskStatusDisable, t.RetryTimes, t.NotifyType, t.MisfirePolicy,
			now, now,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

// updateBuiltinRunbookInstructions 既有内置 Runbook 的指令纠偏：按 key 覆盖为最新模板指令
// （公共引导 + 正文）。仅更新 instruction 列，不碰用户改过的调度/状态等配置。
// 迁移 v1.12.2 引入：此前 seed 按 key 查重跳过，模板指令优化对已落库任务不生效
func updateBuiltinRunbookInstructions(tx *gorm.DB) error {
	for _, t := range BuiltinRunbookTemplates {
		if err := tx.Model(&agentopsentity.AgentTask{}).
			Where("`key` = ?", t.Key).
			Update("instruction", fullInstruction(t)).Error; err != nil {
			return err
		}
	}
	return nil
}

// 展示型列 → Extra 收敛的迁移支撑（v1.12.2）。
//
// 判据见 agentops/domain/entity/extra.go：owner 账号名、备注、反馈备注都是纯展示字段，按
// docs/server/domain.md 不该占独立列。实体已改为「瞬态字段 + Extra 落库」，本步在已建表的库上：
// ① 补出 extra 列（由调用方的 AutoMigrate 完成）；② 把旧列已有值并入 extra（否则升级后历史行的
// 负责人/备注在界面上凭空变空）；③ 删除废弃列。顺序不可颠倒：删列后就没有值可回填。
//
// 全程不拼接列名到 SQL（取行时不指定列、回写时只写 extra 列），避免动态标识符进语句；
// 回填的读写全在 Go 侧完成：MySQL 的 JSON_SET 与 SQLite 的 json_* 签名不一致，
// 而本仓迁移必须同时跑通两种方言（见 sqlite_full_chain_test 与迁移方言交集约束）。

// displayColumnSpec 一条「旧列 → Extra 键」的回填规则
type displayColumnSpec struct {
	dest      any    // 实体：解析真实表名与列信息
	table     string // 表名
	oldColumn string // 待废弃的列名
	extraKey  string // 回填进 Extra 的键名（引用 domain 常量，不另立字面量真源）
}

// agentOpsDisplayColumns 待收敛的展示列清单
func agentOpsDisplayColumns() []displayColumnSpec {
	return []displayColumnSpec{
		{&agentopsentity.Run{}, "t_aiops_run", "owner_username", agentopsentity.ExtraKeyOwnerUsername},
		{&agentopsentity.Run{}, "t_aiops_run", "feedback_note", agentopsentity.ExtraKeyFeedbackNote},
		{&agentopsentity.AgentTask{}, "t_aiops_task", "owner_username", agentopsentity.ExtraKeyOwnerUsername},
		{&agentopsentity.AgentTask{}, "t_aiops_task", "remark", agentopsentity.ExtraKeyRemark},
		{&agentopsentity.Trigger{}, "t_aiops_trigger", "owner_username", agentopsentity.ExtraKeyOwnerUsername},
		{&agentopsentity.Trigger{}, "t_aiops_trigger", "remark", agentopsentity.ExtraKeyRemark},
		{&agentopsentity.Proposal{}, "t_aiops_proposal", "owner_username", agentopsentity.ExtraKeyOwnerUsername},
	}
}

// collapseAgentOpsDisplayColumns 回填旧列值并删除废弃列
func collapseAgentOpsDisplayColumns(tx *gorm.DB) error {
	for _, spec := range agentOpsDisplayColumns() {
		columns, err := existingColumns(tx, spec.dest)
		if err != nil {
			return err
		}
		if !columns[spec.oldColumn] {
			continue // 全新库或已迁移过：无列可回填
		}
		if err := backfillColumnIntoExtra(tx, spec); err != nil {
			return err
		}
		// 不用 Migrator().DropColumn：glebarez/sqlite 在该驱动版本里它对实体不再声明的列是
		// 静默空操作（实测 err==nil 但列仍在），旧列会以 NULL 留在库里并继续被旧代码误读。
		// 裸语句是 MySQL 与 SQLite(>=3.35) 的方言交集，标识符由 clause.Table/Column 交给驱动引用
		if err := tx.Exec("ALTER TABLE ? DROP COLUMN ?",
			clause.Table{Name: spec.table}, clause.Column{Name: spec.oldColumn}).Error; err != nil {
			return err
		}
	}
	return nil
}

// existingColumns 返回表实际存在的列名集合。
//
// 用 Migrator().ColumnTypes 而非实体字段：实体已不再声明这些列，只有库内 schema 才是
// 「列是否存在」的事实源（MySQL/SQLite 均由 Migrator 屏蔽方言差异）
func existingColumns(tx *gorm.DB, dest any) (map[string]bool, error) {
	types, err := tx.Migrator().ColumnTypes(dest)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(types))
	for _, ct := range types {
		out[strings.ToLower(ct.Name())] = true
	}
	return out, nil
}

// backfillColumnIntoExtra 把旧列的非空值并入 extra；已有同名键不覆盖（以实体侧现值为准）。
func backfillColumnIntoExtra(tx *gorm.DB, spec displayColumnSpec) error {
	// 不显式 Select 列：表名与列名均为本文件内的编译期常量，但取全部列后在 Go 侧按列名取值，
	// 可以避开把动态列名写进语句（也免了给不同旧列起别名）
	var rows []map[string]any
	if err := tx.Table(spec.table).Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		oldValue := textValue(row[spec.oldColumn])
		if oldValue == "" {
			continue // 无值可迁：不必写键
		}
		extra := parseExtra(row["extra"])
		if _, ok := extra[spec.extraKey]; ok {
			continue // extra 里已有该键（例如已跑过新代码写入），不覆盖
		}
		extra[spec.extraKey] = oldValue
		if err := tx.Table(spec.table).Where("id = ?", cast.ToUint64(row["id"])).
			UpdateColumn("extra", collx.M(extra)).Error; err != nil {
			return err
		}
	}
	return nil
}

// textValue 列值转文本（驱动可能返回 []byte 或 string，NULL 归空串）
func textValue(val any) string {
	switch v := val.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return cast.ToString(v)
	}
}

// parseExtra 解析行内 Extra（NULL/空/非法 JSON 一律按空容器处理，回填时按对象写回）
func parseExtra(raw any) map[string]any {
	switch v := raw.(type) {
	case nil:
		return map[string]any{}
	case []byte:
		return parseExtraStr(string(v))
	case string:
		return parseExtraStr(v)
	default:
		return map[string]any{}
	}
}

// parseExtraStr 按 JSON 对象解析 extra 文本（空串与非法 JSON 均按空容器处理，回填时重建为对象）
func parseExtraStr(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	parsed, err := jsonx.ToMapByStr(raw)
	if err != nil || parsed == nil {
		return map[string]any{}
	}
	return parsed
}

// 目标资源列 → 指令内引用（v1.12.2）。
//
// 「目标资源」原来是一列手写 JSON，只作为运行输入上下文下发给模型；AI 助手输入框已支持 `@`
// 引用资源（芯片段自带 machineId/dbName 等完整定位信息），再留一列等于同一件事两个入口，
// 而那一列的表单控件是让运维人员直接写 JSON。
//
// 删列前必须把存量值折进指令：只删列会让已有任务在下次运行时「没有目标」，模型转而自己盘点甚至
// 猜测资源——那是无报错的行为改变。折进指令后信息量不变（原来也只是把 JSON 文本喂给模型），
// 变化点只有位置：从 user 输入移到 system 指令，二者都会进同一轮上下文。

// targetResourcesColumn 待收敛的旧列名
const targetResourcesColumn = "target_resources"

// foldAgentOpsTargetResourcesIntoInstruction 把旧列值并入指令文本，再删除该列
func foldAgentOpsTargetResourcesIntoInstruction(tx *gorm.DB) error {
	columns, err := existingColumns(tx, &agentopsentity.AgentTask{})
	if err != nil {
		return err
	}
	if !columns[targetResourcesColumn] {
		return nil // 全新库或已迁移过：无列可折
	}

	// 与展示列收敛同一做法：不指定列取整行，避免把动态列名拼进语句
	var rows []map[string]any
	if err := tx.Table("t_aiops_task").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		targets := textValue(row[targetResourcesColumn])
		if targets == "" {
			continue
		}
		// 措辞沿用原 buildTaskInput 的注入文本，使模型侧读到的语义不变
		merged := textValue(row["instruction"]) + "\n\n目标资源(JSON，其中的 id/resourceId 可直接作为工具入参使用)：\n" + targets
		if err := tx.Table("t_aiops_task").Where("id = ?", cast.ToUint64(row["id"])).
			UpdateColumn("instruction", merged).Error; err != nil {
			return err
		}
	}

	// 不用 Migrator().DropColumn：glebarez/sqlite 对实体不再声明的列是静默空操作（实测 err==nil
	// 但列仍在）；裸语句是 MySQL 与 SQLite(>=3.35) 的方言交集
	return tx.Exec("ALTER TABLE ? DROP COLUMN ?",
		clause.Table{Name: "t_aiops_task"}, clause.Column{Name: targetResourcesColumn}).Error
}

// seedAgentDiagnosisNotifyTmpl 落一条「运维诊断结果通知」模板（按 code 查重，幂等且不覆盖）。
//
// 模板文本引用 agentopsentity.DiagnosisNotifyTmpl，与 notifyResult 下发的参数键同源（见
// entity/diagnosis_notify.go 及其覆盖测试）。code 用固定语义串而非随机值，既作幂等查重基准，
// 也便于文档/引用。刻意不写 t_msg_tmpl_channel：渠道是部署环境特定的，seed 只给内容，
// 管理员在「消息渠道」配好渠道后于模板编辑里关联
func seedAgentDiagnosisNotifyTmpl(tx *gorm.DB) error {
	const code = "agentopsDiagnosisResult"
	var count int64
	if err := tx.Table("t_msg_tmpl").
		Where("code = ? AND is_deleted = 0", code).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // 已存在（含存量环境手工建过同名 code）不覆盖
	}
	now := time.Now()
	return tx.Table("t_msg_tmpl").Create(map[string]any{
		"name":        "运维诊断结果通知",
		"code":        code,
		"title":       "运维诊断结果",
		"tmpl":        agentopsentity.DiagnosisNotifyTmpl,
		"msg_type":    1, // text
		"status":      1, // enable
		"remark":      "系统内置：Agent 运维诊断/触发器/任务的结果通知模板，占位符与运行收尾通知参数同源；需在「消息渠道」配好渠道并为本模板关联后才能真正送达",
		"extra":       "{}",
		"creator":     "system",
		"creator_id":  0,
		"modifier":    "system",
		"modifier_id": 0,
		"create_time": now,
		"update_time": now,
		"is_deleted":  0,
	}).Error
}
