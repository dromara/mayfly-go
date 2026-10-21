package entity

import (
	"time"

	"github.com/spf13/cast"

	"mayfly-go/pkg/model"

	"gorm.io/gorm"
)

// Trigger 事件驱动触发器：把「某来源的某类事件」绑定到「一次 Agent 运行配置」。
//
// 一条触发器 = 一份「何来源（source_type）、命中何条件（match_cond）、跑什么（task_id 复用
// 既有任务 或 内联 instruction/profile）、以何身份（owner）、冷却多久（cooldown_seconds）、
// 是否启用（enabled）」的声明。事件到达 → TriggerService 匹配启用触发器 → 冷却去抖 + 幂等键
// → RunService.Enqueue（trigger=event）产出 Run。
//
// 两种运行配置来源（二选一，task_id 优先）：
//   - 关联任务（task_id>0）：复用任务的 instruction/profile/owner/timeout/retry/notify 全套配置，
//     适合「告警 → 跑对应 Runbook 巡检任务」；事件资源信息并入运行输入。
//   - 内联（task_id=0）：直接给 instruction/profile/owner，运行参数走服务默认，适合 webhook 等
//     无既有任务的轻量接入。
//
// 用 model.Model（含逻辑删除）：触发器是用户可增删改的配置实体，删除留痕与任务口径一致。
type Trigger struct {
	model.Model
	// ExtraData 展示型辅助字段的容器（owner 账号名、备注等不进查询/排序的字段收敛在这里）
	model.ExtraData

	// Name 触发器名称（展示；同时作为派生 Run 的标题前缀）
	Name string `json:"name" form:"name" gorm:"not null;size:255;comment:触发器名称"`

	// SourceType 触发源类型（alert/webhook；见 trigger.SourceType*）
	SourceType string `json:"sourceType" form:"sourceType" gorm:"not null;size:32;index:idx_aiops_trigger_source;comment:触发源类型"`

	// MatchCond 匹配条件 JSON（trigger.MatchCondition；空=命中该来源全部事件）
	MatchCond string `json:"matchCond" gorm:"type:text;comment:匹配条件JSON"`

	// TaskId 关联任务 id（>0 时复用任务全套运行配置；0=用内联字段）
	TaskId uint64 `json:"taskId" form:"taskId" gorm:"index:idx_aiops_trigger_task;comment:关联任务id"`

	// Instruction 内联指令（task_id=0 时用；system 级，引用以可读标记承载，见 AgentTask.Instruction）
	Instruction string `json:"instruction" gorm:"type:text;comment:内联指令"`
	// InstructionSegments 内联指令的芯片引用段（瞬态字段，收敛在 Extra，语义同 AgentTask）
	InstructionSegments []InstructionSegment `json:"instructionSegments" gorm:"-"`
	// ProfileId 内联 Agent 装配档（task_id=0 时用；空默认 diagnostician）
	ProfileId string `json:"profileId" form:"profileId" gorm:"size:64;comment:内联装配档"`

	// NotifyType 内联模式的结果通知渠道（0=不通知）。瞬态字段：落库收敛在 Extra。
	//
	// 没有它，内联触发器（webhook/告警的主用法）派发的诊断运行永远不通知任何人：
	// resolveRunConfig 只在「关联任务」分支从任务取 notify*，内联分支只能从触发器自身取
	NotifyType int8 `json:"notifyType" form:"notifyType" gorm:"-"`
	// NotifyTmplCode 结果通知的消息模板码（瞬态字段，同 NotifyType 走 Extra）
	NotifyTmplCode string `json:"notifyTmplCode" form:"notifyTmplCode" gorm:"-"`

	// OwnerAccountId 运行服务身份账号 id（内联模式必填；关联任务时用任务 owner）
	OwnerAccountId uint64 `json:"ownerAccountId" form:"ownerAccountId" gorm:"comment:运行身份账号id"`
	// OwnerUsername 运行服务身份账号名（展示/审计）。瞬态字段：落库收敛在 Extra（非查询维度）
	OwnerUsername string `json:"ownerUsername" gorm:"-"`

	// CooldownSeconds 冷却去抖秒数（同一触发器两次派生运行的最小间隔；0=不冷却）
	CooldownSeconds int `json:"cooldownSeconds" form:"cooldownSeconds" gorm:"comment:冷却秒数"`

	// WebhookToken webhook 专用推送令牌（source_type=webhook 时用于 URL 路径校验）：
	// 只由服务端以 CSPRNG 生成、不采信客户端输入，更新时保留旧值、轮换走显式动作。
	// 索引刻意非唯一：非 webhook 来源的行共享空串，建唯一索引会拦下第二条普通触发器；
	// 唯一性由应用层查重保证（服务端随机取值，碰撞概率可忽略）
	WebhookToken string `json:"webhookToken" gorm:"size:64;index:idx_aiops_trigger_token;comment:webhook令牌"`

	// Enabled 是否启用（1启用 -1禁用；见 TriggerEnabled/TriggerDisabled）
	Enabled int8 `json:"enabled" form:"enabled" gorm:"not null;default:1;index:idx_aiops_trigger_enabled;comment:是否启用"`

	// LastFireTime 上次派生运行时间（冷却判定基线）
	LastFireTime *time.Time `json:"lastFireTime" gorm:"comment:上次派生时间"`

	// Remark 备注（瞬态字段，收敛进 Extra）
	Remark string `json:"remark" gorm:"-"`
}

// BeforeSave 把展示型瞬态字段收进 Extra（Extra 是唯一落库真源），钩子形参说明见 AgentTask.BeforeSave
func (t *Trigger) BeforeSave(*gorm.DB) error {
	t.SetExtraValue(ExtraKeyOwnerUsername, t.OwnerUsername)
	t.SetExtraValue(ExtraKeyRemark, t.Remark)
	t.SetExtraValue(ExtraKeyNotifyType, t.NotifyType)
	t.SetExtraValue(ExtraKeyNotifyTmplCode, t.NotifyTmplCode)
	setInstructionSegments(&t.ExtraData, t.InstructionSegments)
	return nil
}

// AfterFind 从 Extra 还原瞬态展示字段（VO 按名拷贝与 JSON 输出保持原契约）
func (t *Trigger) AfterFind(*gorm.DB) error {
	t.OwnerUsername = t.GetExtraString(ExtraKeyOwnerUsername)
	t.Remark = t.GetExtraString(ExtraKeyRemark)
	// JSON 容器里的数字解回来是 float64，靠 cast 统一转 int8，不在这里手写类型断言
	t.NotifyType = cast.ToInt8(t.GetExtraVal(ExtraKeyNotifyType))
	t.NotifyTmplCode = t.GetExtraString(ExtraKeyNotifyTmplCode)
	t.InstructionSegments = instructionSegments(&t.ExtraData)
	return nil
}

// TableName 指定表名
func (t *Trigger) TableName() string {
	return "t_aiops_trigger"
}

// 触发器启用状态（与 AgentTaskStatus* 同口径，便于前端复用枚举）
const (
	TriggerEnabled  int8 = 1  // 启用（参与事件匹配）
	TriggerDisabled int8 = -1 // 禁用（不参与匹配）
)
