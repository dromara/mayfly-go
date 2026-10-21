package protocol

import (
	"strings"

	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
)

// 中断持久化契约
//
// 中断持久化结构（InterruptInfo / InterruptResume）与 SSE 事件层
//（Interrupt）分层：本文件对应持久化层——tool_call item extra["interrupt"]
// 的强类型结构，写入侧（挂起时构建落库）与读取侧（resume 回写决策、前端
// 历史还原）共用同一形状；WS 事件层仍由 InterruptEvent 承载。

// ExtraKeyInterrupt tool_call item 的 extra 列承载中断信息的键（唯一真源）。
// 写入侧（turn_recorder 挂起、session_store/turn_item resume 回写）与读取侧（interruptInfoOf）共用
const ExtraKeyInterrupt = "interrupt"

// InterruptInfo tool_call item extra["interrupt"] 的强类型结构。
// 写入侧（挂起时 NewInterruptInfo 构建后序列化落库）与读取侧（resume 写决策、
// 前端历史还原）共用同一形状，消除手写 JSON key 的拼写风险
type InterruptInfo struct {
	// 中断类型短名（approval / param_completion，内部标签 kind；
	// Go 无 serde flatten 带数据枚举，类型携带的数据统一放 Metadata）
	Kind string `json:"kind"`
	// 中断请求唯一 ID
	RequestId string `json:"request_id"`
	// 人类可读的中断原因
	Message string `json:"message"`
	// 会话 ID（前端构建恢复请求用）
	ConversationId uint64 `json:"conversation_id"`
	// Agent ID（mayfly-go 当前为单 Agent 进程，保留字段）
	AgentId int64 `json:"agent_id"`
	// 中断类型携带的数据（#[serde(flatten)] kind 变体数据：
	// payload/title/missingFields/options 等，前端历史还原中断卡用）
	Metadata map[string]any `json:"metadata,omitempty"`
	// 工具自声明的记忆策略（RememberPolicy，mayfly-go 暂未启用决策缓存）
	Remember *RememberPolicy `json:"remember,omitempty"`
	// 恢复决策（挂起时为 nil，resume 后经 UpdateMessage 回写）
	Resume *InterruptResume `json:"resume,omitempty"`
}

// RememberPolicy 工具自声明的记忆策略
type RememberPolicy struct {
	// 自定义指纹材料（引擎叠加 tool_name 作命名空间，工具间不互相命中）
	Fingerprint string `json:"fingerprint,omitempty"`
	// 允许的最大记住范围（turn / conversation ...）
	MaxScope string `json:"max_scope,omitempty"`
}

// InterruptResume 中断恢复决策（extra["interrupt"]["resume"]，
// tag "type" snake_case，approved/rejected/answered/skipped/
// params_completed；各决策的差异字段以可选字段承载）
type InterruptResume struct {
	// 决策类型：approved | rejected | answered | skipped | params_completed
	Type string `json:"type"`
	// rejected 时的拒绝原因
	Reason string `json:"reason,omitempty"`
	// answered 时的用户回答
	Answer string `json:"answer,omitempty"`
	// 恢复参数（mayfly-go 扩展：参数补全/审批恢复链路所需的 payload）
	Payload collx.M `json:"payload,omitempty"`
}

// InterruptKindOf LLM 消息类型转中断类型短名（interrupt_approval → approval）
func InterruptKindOf(msgType string) string {
	return strings.TrimPrefix(msgType, "interrupt_")
}

// InterruptMsgType 中断类型短名转恢复链路的 LLM 消息类型（approval → interrupt_approval）
func InterruptMsgType(kind string) string {
	return "interrupt_" + kind
}

// 恢复动作与 resume 决策类型：跨前后端线协议的取值，唯一真源。
//
// 为什么必须常量化：同一组值曾在三处手写——ResumeTypeOfAction 的映射、审批恢复的 switch、
// 参数补全的 paramActionComplete。新增一种决策时改一处不会报错，漏两处就静默走默认分支
// （审批变成「无效动作」、补全不被识别为已完成）
const (
	ActionApprove  = "approve"
	ActionReject   = "reject"
	ActionComplete = "complete"

	ResumeTypeApproved        = "approved"
	ResumeTypeRejected        = "rejected"
	ResumeTypeParamsCompleted = "params_completed"
)

// ResumeTypeOfAction 恢复动作转 resume 决策类型（approve → approved，snake_case）
func ResumeTypeOfAction(action string) string {
	switch action {
	case ActionApprove:
		return ResumeTypeApproved
	case ActionReject:
		return ResumeTypeRejected
	case ActionComplete:
		return ResumeTypeParamsCompleted
	default:
		return action
	}
}

// ResumeActionOfType resume 决策类型转恢复动作（approved → approve）
func ResumeActionOfType(resumeType string) string {
	switch resumeType {
	case ResumeTypeApproved:
		return ActionApprove
	case ResumeTypeRejected:
		return ActionReject
	case ResumeTypeParamsCompleted:
		return ActionComplete
	default:
		return resumeType
	}
}

// NewInterruptInfo 从 WS 事件层的 InterruptEvent 构建挂起持久化结构
// （InterruptInfo::from_reason，resume 挂起时为空）
func NewInterruptInfo(evt *InterruptEvent, conversationId uint64) *InterruptInfo {
	if evt == nil {
		return nil
	}
	return &InterruptInfo{
		Kind:           InterruptKindOf(evt.Type),
		RequestId:      evt.ActionId,
		Message:        evt.Description,
		ConversationId: conversationId,
		Metadata:       evt.Metadata,
	}
}

// NewResumeFromResumeInfo 将恢复链路回写的 resumeInfo（tools.InterruptResume 形状）
// 转为持久化 resume 决策（决策内嵌 interrupt 对象，merge_patch 语义）
func NewResumeFromResumeInfo(resumeInfo any) *InterruptResume {
	mp, err := jsonx.ToByStr[map[string]any](jsonx.ToStr(resumeInfo))
	if err != nil || mp == nil {
		return nil
	}
	m := collx.M(*mp)
	r := &InterruptResume{
		Type:   ResumeTypeOfAction(m.GetStr("action")),
		Reason: m.GetStr("reason"),
	}
	if payload := m["payload"]; payload != nil {
		if pm, err := jsonx.ToByStr[map[string]any](jsonx.ToStr(payload)); err == nil && pm != nil {
			r.Payload = collx.M(*pm)
		}
	}
	return r
}

// ToResumeInfo 将持久化 resume 决策反合成为恢复链路消费的 resumeInfo
// （tools.InterruptResume 形状：turnId/interruptId/interruptType/action/payload）
func (r *InterruptResume) ToResumeInfo(kind, requestId, turnId string) collx.M {
	return collx.M{
		"turnId":        turnId,
		"interruptId":   requestId,
		"interruptType": InterruptMsgType(kind),
		"action":        ResumeActionOfType(r.Type),
		"payload":       r.Payload,
	}
}
