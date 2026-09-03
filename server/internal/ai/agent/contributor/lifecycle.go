package contributor

import (
	"context"
)

// TurnEndCause 轮次结束原因（对齐 tokhub TurnEndInput::cause，按因施策）
type TurnEndCause string

const (
	// TurnEndCompleted 正常完成
	TurnEndCompleted TurnEndCause = "completed"
	// TurnEndError 异常终止
	TurnEndError TurnEndCause = "error"
	// TurnEndAborted 主动中止（用户停止）
	TurnEndAborted TurnEndCause = "aborted"
	// TurnEndInterrupted 中断挂起（等待用户决策后恢复）
	TurnEndInterrupted TurnEndCause = "interrupted"
)

// TurnStartInput 轮次开始输入
type TurnStartInput struct {
	SessionKey string
	TurnId     string
	UserId     string
}

// TurnEndInput 轮次结束输入
type TurnEndInput struct {
	SessionKey string
	TurnId     string
	UserId     string
	// Cause 结束原因：正常完成 / 异常终止 / 主动中止 / 中断挂起
	Cause TurnEndCause
}

// TurnLifecycleContributor 轮次生命周期贡献者
//
// 在轮次开始和结束时被调用。典型用途：
//   - OnTurnStart：初始化轮次级配置
//   - OnTurnEnd：清理临时数据、记录统计指标，按 Cause 区分正常完成 /
//     异常终止 / 主动中止 / 中断挂起，按因施策
//
// 实现应无状态或自行保证并发安全（每轮生命周期都会调用）。
// 对齐 tokhub TurnLifecycleContributor。
type TurnLifecycleContributor interface {
	Contributor
	// OnTurnStart 轮次开始时调用
	OnTurnStart(ctx context.Context, in *TurnStartInput) error
	// OnTurnEnd 轮次结束时调用
	OnTurnEnd(ctx context.Context, in *TurnEndInput) error
}
