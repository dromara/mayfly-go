package form

import "mayfly-go/internal/agentops/domain/entity"

// AgentTaskForm 定时计划任务表单（创建/更新）。
//
// 与 entity.AgentTask 分列：表单只暴露用户可编辑字段，屏蔽 key（服务端生成不可改）、
// lastFireTime（触发时由服务端记录）等内部状态，避免客户端篡改运行标识与 misfire 基线。
// ownerAccountId 可空——为空时以当前登录账号为运行身份；指定他人账号需超管或专门权限点（见 resolveRunOwner）。
type AgentTaskForm struct {
	Id          uint64 `json:"id"`
	Name        string `json:"name" binding:"required"`
	Instruction string `json:"instruction" binding:"required"`
	// InstructionSegments 指令内的芯片引用段（技能/资源定位信息，见 entity.InstructionSegment）
	InstructionSegments []entity.InstructionSegment `json:"instructionSegments"`
	Cron                string                      `json:"cron"`
	ProfileId           string                      `json:"profileId"`
	OwnerAccountId      uint64                      `json:"ownerAccountId"`

	Status         int8   `json:"status"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	RetryTimes     int8   `json:"retryTimes"`
	MaxStep        int    `json:"maxStep"`
	TokenBudget    int64  `json:"tokenBudget"`
	DryRun         bool   `json:"dryRun"`
	NotifyType     int8   `json:"notifyType"`
	NotifyTmplCode string `json:"notifyTmplCode"`
	MisfirePolicy  int8   `json:"misfirePolicy"`
	Remark         string `json:"remark"`
}

// TriggerForm 事件驱动触发器表单（创建/更新）。
//
// 与 entity.Trigger 分列：表单只暴露用户可编辑字段，屏蔽 lastFireTime（冷却基线，服务端
// 派发时记录）、ownerUsername（服务端根据运行身份解析）等内部状态。
// 运行配置二选一：taskId>0 复用关联任务配置（无需 owner/instruction）；否则内联 instruction+ownerAccountId。
//
// 此处刻意没有 webhookToken 字段：推送令牌只由服务端生成、更新时保留旧值（轮换走
// rotateWebhookToken 显式动作）。客户端传入令牌会留下面向任何可保存触发器的账号「钉住一个已知令牌」
// 或猜中他人令牌的入口。
type TriggerForm struct {
	Id         uint64 `json:"id"`
	Name       string `json:"name" binding:"required"`
	SourceType string `json:"sourceType" binding:"required"`
	MatchCond  string `json:"matchCond"`

	TaskId      uint64 `json:"taskId"`
	Instruction string `json:"instruction"`
	// InstructionSegments 指令的芯片引用段（语义同 AgentTaskForm）：不收则前端采到了也没地方放，
	// 派发时只能给模型一句带字面 `@名字` 的纯文本，它不知道要操作哪台机器/哪个库
	InstructionSegments []entity.InstructionSegment `json:"instructionSegments"`
	ProfileId           string                      `json:"profileId"`
	// NotifyType/NotifyTmplCode 内联模式的结果通知配置（关联任务时以任务为准）；
	// 不收则「外部告警进来→自动诊断→通知到人」这条核心链路最后一跳不通
	NotifyType     int8   `json:"notifyType"`
	NotifyTmplCode string `json:"notifyTmplCode"`

	OwnerAccountId uint64 `json:"ownerAccountId"`
	// RotateWebhookToken true=显式轮换推送令牌（仅 webhook 来源生效），旧地址随即失效
	RotateWebhookToken bool   `json:"rotateWebhookToken"`
	CooldownSeconds    int    `json:"cooldownSeconds"`
	Enabled            int8   `json:"enabled"`
	Remark             string `json:"remark"`
}
