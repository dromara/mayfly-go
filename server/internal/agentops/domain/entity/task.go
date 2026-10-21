package entity

import (
	"time"

	"mayfly-go/pkg/model"

	"gorm.io/gorm"
)

// AgentTask 定时/计划触发的 Agent 任务（对标 machine_cronjob，但驱动的是无头 Agent 运行）。
//
// 一条任务 = 一份「何时（cron）、以何身份（owner）、用何档（profile）、做什么（instruction）、
// 对哪些资源（target_resources）、跑多久（timeout/max_step）、失败怎么办（retry）、结果通知谁
// （notify）」的声明。到点由 cron 触发 → RunByKey → RunService.Enqueue(trigger=cron) 产出 Run。
//
// 用 model.Model（含逻辑删除）：任务是用户可增删改的配置实体，删除留痕与机器计划任务口径一致。
type AgentTask struct {
	model.Model
	// ExtraData 展示型辅助字段的容器（owner 账号名、备注等不进查询/排序的字段都收敛在这里）
	model.ExtraData

	// Name 任务名称（展示）
	Name string `json:"name" form:"name" gorm:"not null;size:255;comment:任务名称"`
	// Key 任务唯一 key（cron 绑定/解绑、RunByKey 定位、Run.trigger_ref 均用它；创建时生成不可改）
	Key string `json:"key" gorm:"not null;size:64;uniqueIndex:uk_aiops_task_key;comment:任务key"`

	// Instruction 任务指令（system 级）。文本里以可读标记承载引用（技能 `$code`、资源 `@名称`），
	// 可直接取参的定位信息由瞬态字段 InstructionSegments 一并保存，派发时展开给模型
	Instruction string `json:"instruction" gorm:"type:text;comment:任务指令"`
	// InstructionSegments 指令内的芯片引用段（技能/资源）。瞬态字段：收敛在 Extra（非查询维度），
	// 由 BeforeSave/AfterFind 与 Extra 双向同步
	//
	// 为什么不能只靠文本：模型看到「@aliyun」并不能直接调工具——机器工具要 machineId/ip/port、
	// 数据库工具要 dbId/dbName。段里带的就是这些定位信息，派发时按聊天消息同一套渲染器展开
	InstructionSegments []InstructionSegment `json:"instructionSegments" gorm:"-"`
	// Cron cron 表达式（5/6 字段皆可，见 pkg/scheduler 解析器；禁用态或空则不调度）
	Cron string `json:"cron" form:"cron" gorm:"size:255;comment:cron表达式"`

	// ProfileId Agent 装配档（空则默认 diagnostician 只读诊断档）
	ProfileId string `json:"profileId" form:"profileId" gorm:"size:64;comment:Agent装配档"`

	// OwnerAccountId 运行服务身份账号 ID（必填；工具权限过滤按此收敛，fail-closed）
	OwnerAccountId uint64 `json:"ownerAccountId" form:"ownerAccountId" gorm:"not null;comment:运行身份账号id"`
	// OwnerUsername 运行服务身份账号名（展示/审计）。瞬态字段：落库收敛在 Extra，
	// 由 BeforeSave/AfterFind 钩子与 Extra 双向同步，不建独立列（非查询维度）
	OwnerUsername string `json:"ownerUsername" gorm:"-"`

	// Status 状态（1启用 -1禁用；见 AgentTaskStatus* 常量）
	Status int8 `json:"status" form:"status" gorm:"comment:状态"`

	// TimeoutSeconds 单次运行超时秒数（0=用服务默认）
	TimeoutSeconds int `json:"timeoutSeconds" gorm:"comment:单次运行超时秒数"`
	// RetryTimes 失败重试次数（仅超时/连接类重试；0=不重试）
	RetryTimes int8 `json:"retryTimes" gorm:"comment:失败重试次数"`
	// MaxStep 单次运行最大步数（0=用 Profile 默认）
	MaxStep int `json:"maxStep" gorm:"comment:最大步数"`

	// TokenBudget 单次运行 token 预算熔断（0=不熔断；>0 时快照到 Run 并在运行期强制执行）
	TokenBudget int64 `json:"tokenBudget" gorm:"comment:token预算熔断"`

	// DryRun 影子模式（演练）：触发的运行只读执行，仅产出「将执行什么」不落地任何变更，
	// 用于安全验证任务指令/编排逻辑（executor 档在影子模式下由 RunService 强制降为只读档）
	DryRun bool `json:"dryRun" gorm:"comment:影子模式"`

	// NotifyType 结果通知方式（见 TaskNotify* 常量）
	NotifyType int8 `json:"notifyType" gorm:"comment:结果通知方式"`
	// NotifyTmplCode 结果通知消息模板 code（渠道由模板关联决定）
	NotifyTmplCode string `json:"notifyTmplCode" gorm:"size:64;comment:通知模板code"`

	// MisfirePolicy 错过触发策略（进程重启期间错过的窗口如何处理；见 MisfirePolicy* 常量）
	MisfirePolicy int8 `json:"misfirePolicy" gorm:"comment:错过触发策略"`
	// LastFireTime 上次实际触发时间（misfire 判定基线；in-memory 调度器不持久化下次触发点，
	// 故以「上次触发时间 + cron 下一次触发」推断重启期间是否错过窗口）
	LastFireTime *time.Time `json:"lastFireTime" gorm:"comment:上次触发时间"`

	// Remark 备注（瞬态字段，同上，收敛进 Extra）
	Remark string `json:"remark" gorm:"-"`
}

// BeforeSave 把展示型瞬态字段收进 Extra（Extra 是唯一落库真源）。
//
// 无独立列后若不在这里写入，赋值到字段上的值会静默丢失（GORM 不映射 gorm:"-"）；
// 钩子必须带 *gorm.DB 形参：GORM 1.31 不调用无参签名的钩子（已实测）
func (t *AgentTask) BeforeSave(*gorm.DB) error {
	t.SetExtraValue(ExtraKeyOwnerUsername, t.OwnerUsername)
	t.SetExtraValue(ExtraKeyRemark, t.Remark)
	setInstructionSegments(&t.ExtraData, t.InstructionSegments)
	return nil
}

// AfterFind 从 Extra 还原瞬态展示字段，供 VO 按名拷贝与 JSON 输出（接口契约不变）
func (t *AgentTask) AfterFind(*gorm.DB) error {
	t.OwnerUsername = t.GetExtraString(ExtraKeyOwnerUsername)
	t.Remark = t.GetExtraString(ExtraKeyRemark)
	t.InstructionSegments = instructionSegments(&t.ExtraData)
	return nil
}

// TableName 指定表名
func (a *AgentTask) TableName() string {
	return "t_aiops_task"
}

// AuditUpdateColumns 走显式列更新时必须一并列入的审计列。
//
// 为什么需要写定：GORM 只会自动维护 create_time/update_time 这类约定时间列，而 modifier/modifier_id
// 是基座 FillBaseInfo 设在实体上的普通字段；一旦更新走了 Select(指定列) 而未列入它们，修改人就会
// 静默停留在创建者（实测：bob 改完后 modifier 仍为 alice），「谁改了这条运维配置」的审计链断掉。
var AuditUpdateColumns = []string{"modifier", "modifier_id", "update_time"}

// TaskEditableColumns 任务表单可编辑、且保存时必须整列真实写入的列集合（DB 列名）。
//
// 为什么不用结构体更新（App.Save/UpdateById 不带列名）：GORM 结构体更新跳过零值字段，
// 而本实体的「有意义取值」里大量包含零值——dry_run=false（关影子模式）、notify_type=0
// （取消通知）、misfire_policy=0（改回跳过）、cron=”（停止调度）、
// max_step/timeout_seconds/token_budget=0（改回服务默认）。
// 走结构体更新会让这些「关/清/改回默认」的操作静默不生效（保存成功但库里仍是旧值），
// 对以 owner 身份驱动真实资源的运维任务而言是行为失真而非显示问题。
//
// 刻意排除服务端独占列：key（创建时生成、不可改）、last_fire_time（触发时记录，
// 表单不携带；若写回会用陈旧零值清掉 misfire/冷却判定基线）。
//
// owner_username / remark 不在列表：它们不再是列，而是由 BeforeSave 钩子写进 extra，
// 故这里列入 extra 才能随保存落库（调用方需先继承库中现值 Extra 再写，避免覆盖其他键）。
var TaskEditableColumns = []string{
	"name", "instruction", "cron", "profile_id",
	"owner_account_id", "status",
	"timeout_seconds", "retry_times", "max_step", "token_budget", "dry_run",
	"notify_type", "notify_tmpl_code", "misfire_policy", "extra",
}

// TriggerEditableColumns 触发器表单可编辑、保存时必须整列真实写入的列集合（DB 列名）。
//
// 语义与约束同 TaskEditableColumns；额外包含 webhook_token —— 该列由应用层先解析出
// 「本次保存后应生效的令牌」（更新时保留旧值或显式轮换）再写入，因此可以整列写回；
// 表单直接传来的令牌值不会被采信。刻意排除 last_fire_time（冷却判定基线，服务端记录）。
// owner_username / remark 同样已收敛进 extra（由 BeforeSave 钩子写入，调用方需先继承旧 Extra）。
var TriggerEditableColumns = []string{
	"name", "source_type", "match_cond", "task_id", "instruction", "profile_id",
	"owner_account_id", "cooldown_seconds", "webhook_token",
	"enabled", "extra",
}

// 任务状态
const (
	AgentTaskStatusEnable  int8 = 1  // 启用（会被调度）
	AgentTaskStatusDisable int8 = -1 // 禁用（只解绑不调度）
)

// 结果通知方式（Run 与 AgentTask 共用；定义于此作为规范出处）
const (
	TaskNotifyNone   int8 = 0 // 不通知
	TaskNotifyFail   int8 = 1 // 仅失败时通知
	TaskNotifyAlways int8 = 2 // 总是通知
)

// misfire（错过触发）策略
//
// in-memory 调度器（robfig/cron）不持久化「下次触发时间」，进程重启期间到点的触发会丢失。
// 重启后 Init 扫描启用任务时，以 LastFireTime 为基线用 cron 计算「本应触发的下一次时间」，
// 若该时间已过去（说明重启期间错过），按策略处理：
const (
	MisfirePolicySkip        int8 = 0 // 跳过：不补跑，仅等待下一个正常触发点（默认）
	MisfirePolicyCatchUpOnce int8 = 1 // 补跑一次：错过窗口只补跑一次（不逐次补齐，防重启风暴）
)
