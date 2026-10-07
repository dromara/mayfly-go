package application

import (
	"context"
	"errors"
	dbimsg "mayfly-go/internal/db/imsg"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNeedApprovalErrorCodeIsRaised 需提单的错误必须带 CodeNeedWorkTicket。
//
// 前端靠这个码决定拦截处要不要给「提交工单」入口，普通业务失败（含「已被禁止执行」）
// 都必须保持默认码，否则会给出一条批了也不会执行的操作一个提单按钮
func TestNeedApprovalErrorCodeIsRaised(t *testing.T) {
	err := newApprovalError(context.Background(), imsg.ErrTriggerPolicyInvalid, "probe")
	require.True(t, IsNeedApprovalError(err))

	var bizErr *errorx.BizError
	require.True(t, errors.As(err, &bizErr))
	require.Equal(t, CodeNeedWorkTicket, bizErr.Code())
}

func TestOtherFailuresAreNotNeedApproval(t *testing.T) {
	for _, err := range []error{
		nil,
		errors.New("boom"),
		errorx.NewBiz("forbidden by policy"),
		errorx.NewBizCode(501, "token error"),
	} {
		require.False(t, IsNeedApprovalError(err), "错误 %v 不应被判为需提单", err)
	}
}

// TestPreserveNeedApprovalCode 覆盖 api 层的真实陷阱：biz.ErrIsNil 会把任何错误
// 重新包成 code=400，4001 一旦丢失，被拦命令旁边的提单入口就静默消失（界面上看不出任何异常）。
// 所以需提单的错误必须在断言之前原样抛出，其余错误交回常规断言处理
func TestPreserveNeedApprovalCode(t *testing.T) {
	approvalErr := newApprovalError(context.Background(), imsg.ErrTriggerPolicyInvalid, "probe")

	recovered := recoverPanicked(func() { PreserveDecisionCode(approvalErr) })
	require.NotNil(t, recovered, "需提单的错误必须中断请求")
	require.True(t, IsNeedApprovalError(recovered), "中断时必须保留原错误码, got %v", recovered)

	// 已被禁止执行等普通业务失败不该被这里接管：它们仍走 biz.ErrIsNil 的默认路径
	for _, err := range []error{nil, errors.New("boom"), errorx.NewBiz("forbidden by policy")} {
		require.Nil(t, recoverPanicked(func() { PreserveDecisionCode(err) }))
	}
}

// TestPreserveKeepsTheWarnAckCode 「需确认」与「需提单」共用同一个保码接缝。
//
// Redis 命令台的第一版实测就栽在这里：接缝只认 4001，4002 被 biz.ErrIsNil 包成 400，
// 前端 isWarnAckError 判不出，确认框再也不出现——提示写着「可直接执行」，但命令没执行、
// 也没有任何执行入口，用户只能反复点执行
func TestPreserveKeepsTheWarnAckCode(t *testing.T) {
	warnErr := newWarnAckError(context.Background(), 0, "命中提醒")
	recovered := recoverPanicked(func() { PreserveDecisionCode(warnErr) })
	require.NotNil(t, recovered, "需确认的错误必须中断请求")
	require.True(t, IsWarnAckError(recovered), "中断时必须保留确认码, got %v", recovered)
}

// TestBizErrIsNilDropsTheCode 把「为什么需要这个 helper」固定成可执行的证据：
// biz.ErrIsNil 会重新包一个默认码的错误，需提单的判定就此失效。
// 哪天它自己保住了错误码，本测试会失败并提醒删掉这层保护
func TestBizErrIsNilDropsTheCode(t *testing.T) {
	approvalErr := newApprovalError(context.Background(), imsg.ErrTriggerPolicyInvalid, "probe")
	recovered := recoverPanicked(func() { biz.ErrIsNil(approvalErr) })
	require.NotNil(t, recovered)
	require.False(t, IsNeedApprovalError(recovered), "ErrIsNil 的包装丢掉了错误码，这正是必须提前抛出的原因")
}

func recoverPanicked(fn func()) (recovered error) {
	defer func() {
		if value := recover(); value != nil {
			recovered, _ = value.(error)
		}
	}()
	fn()
	return
}

// TestNewBlockErrorIsTheSingleBlockSemantics 三个资源入口共用的阻断语义必须有测试守护。
//
// 分支散在三处时，「禁止执行要不要带命中原因」「需审批的错误码会不会漏」各自漂移过：
// 数据库与 Redis 曾只给一句「已被禁止」，操作者看不出踩了哪条规则
func TestNewBlockErrorIsTheSingleBlockSemantics(t *testing.T) {
	ctx := context.Background()
	required := &trigger.Decision{Severity: flowentity.SeverityRequired, Findings: []*trigger.Finding{{
		Source: "check[db.dml-without-where]", Severity: flowentity.SeverityRequired, Title: "flow.check.x",
		Summary: imsg.TriggerReasonCustomCondition,
	}}}
	forbidden := &trigger.Decision{Severity: flowentity.SeverityForbidden, Findings: []*trigger.Finding{{
		Source: "check[db.destructive-ddl]", Severity: flowentity.SeverityForbidden, Title: "flow.check.x",
		Summary: imsg.TriggerReasonCustomCondition,
	}}}
	// 只关心阻断语义：提醒的三种走向由 TestWarnAckNeedsConfirmation 单独守护
	askSpeech := BlockSpeech{Forbidden: imsg.ErrOperationForbidden, Approval: dbimsg.ErrNeedSubmitWorkTicket, RequireWarnAck: true}

	// 需审批：必须带「需提单」错误码，前端据此才给提单入口
	approvalErr := NewBlockError(ctx, required, askSpeech, false)
	require.Error(t, approvalErr)
	require.True(t, IsNeedApprovalError(approvalErr))
	require.Contains(t, approvalErr.Error(), "自定义", "需审批提示必须带上命中原因")

	// 禁止执行：同一个函数也要把原因带出来，而不是只剩一句「已被禁止」
	forbiddenErr := NewBlockError(ctx, forbidden, askSpeech, false)
	require.Error(t, forbiddenErr)
	require.False(t, IsNeedApprovalError(forbiddenErr), "被禁止的操作提单也不会执行，不能给提单入口")
	require.Contains(t, forbiddenErr.Error(), "自定义", "禁止执行提示必须带上命中原因")

	require.NoError(t, NewBlockError(ctx, nil, askSpeech, false))

	// 没有审批通道的场景（Approval=0）：出现需审批结论按禁止执行拒绝，绝不静默放行
	noChannelErr := NewBlockError(ctx, required, BlockSpeech{Forbidden: imsg.ErrOperationForbidden}, false)
	require.Error(t, noChannelErr, "落不了地的「需审批」必须被拒而不是放行")
	require.False(t, IsNeedApprovalError(noChannelErr))

	// 原因为空时不能留下一个空括号
	bracketless := NewBlockError(ctx, &trigger.Decision{Severity: flowentity.SeverityForbidden}, BlockSpeech{Forbidden: imsg.ErrOperationForbidden}, false)
	require.Error(t, bracketless)
	require.NotContains(t, bracketless.Error(), "（）", "没有原因时不应拼出空括号")
}

// TestWarnAckNeedsConfirmation 「仅提醒」命中时的三种走向必须互不混淆：
// 能确认的入口先要一次确认（此时命令还没执行，转审批才不会跑第二遍），
// 确认后放行，不能确认的入口保持「不阻断、只回显」。
// 把 RequireWarnAck 判反是最危险的改法：提醒级别会一夜之间变成阻断，
// 而调用方只是没传一个布尔值，界面上看不出任何异常
func TestWarnAckNeedsConfirmation(t *testing.T) {
	ctx := context.Background()
	warning := &trigger.Decision{Severity: flowentity.SeverityWarning, Findings: []*trigger.Finding{{
		Source: "check[db.dml-without-where]", Severity: flowentity.SeverityWarning, Title: "flow.check.x",
		Summary: imsg.TriggerReasonCustomCondition,
	}}}
	askSpeech := BlockSpeech{Forbidden: imsg.ErrOperationForbidden, Approval: dbimsg.ErrNeedSubmitWorkTicket, RequireWarnAck: true}

	unacked := NewBlockError(ctx, warning, askSpeech, false)
	require.Error(t, unacked, "命中提醒且未确认时必须要求确认")
	require.True(t, IsWarnAckError(unacked), "确认必须有专属错误码，前端据此才能分流到确认框")
	require.False(t, IsNeedApprovalError(unacked), "确认码不是提单码，两者后续动作完全不同")
	require.Contains(t, unacked.Error(), "自定义", "确认提示要带上命中原因，否则操作者不知道在确认什么")

	// 操作者已选「直接执行」：同一份结论就此放行，提醒随结果回显
	require.NoError(t, NewBlockError(ctx, warning, askSpeech, true))

	// 不要求确认的入口（批量执行、key 面板）行为不变：只回显，不阻断
	require.NoError(t, NewBlockError(ctx, warning, BlockSpeech{Forbidden: imsg.ErrOperationForbidden}, false))

	// 级别比提醒更严时，确认与否都不改变结论：该拦还得拦，不能因为「操作者想直接执行」就放过危险语句
	forbidden := &trigger.Decision{Severity: flowentity.SeverityForbidden, Findings: []*trigger.Finding{{
		Severity: flowentity.SeverityForbidden, Summary: imsg.TriggerReasonCustomCondition,
	}}}
	for _, acked := range []bool{false, true} {
		require.Error(t, NewBlockError(ctx, forbidden, askSpeech, acked), "已确认提醒不能顺带放过禁止执行")
	}
}

// TestWarnNoticeOnlyForWarnings 机器终端的回显文本只在有提醒时出现。
// 返回空串以外的内容会让终端多打一行无关提示，把「没有命中任何规则」也说得像命中了
func TestWarnNoticeOnlyForWarnings(t *testing.T) {
	ctx := context.Background()
	require.Empty(t, WarnNotice(ctx, nil))
	require.Empty(t, WarnNotice(ctx, &trigger.Decision{Severity: flowentity.SeverityDisabled}), "没有提醒就不该有回显")

	notice := WarnNotice(ctx, &trigger.Decision{Severity: flowentity.SeverityWarning, Findings: []*trigger.Finding{{
		Severity: flowentity.SeverityWarning, Summary: imsg.TriggerReasonCustomCondition,
	}}})
	require.Contains(t, notice, "自定义", "回显必须带上命中的规则原因")
	require.NotContains(t, notice, "（）", "没有原因时不能拼出空括号")
}

// TestBizOperatorContextUsesTicketCreator 审批回放必须以工单发起人身份执行。
//
// 回放发生在最后一位审批人点通过的那一刻，上下文里是审批人身份：审批人常常只有审批权、
// 没有那台资源的生产运维权限，用他的身份过资源鉴权会直接失败（工单显示「处理失败」）；
// 即使他有权限，执行归属也记错了人
func TestBizOperatorContextUsesTicketCreator(t *testing.T) {
	type traceKey struct{}
	ctx := context.WithValue(context.Background(), traceKey{}, "trace-1")
	ctx = contextx.WithLoginAccount(ctx, &model.LoginAccount{Id: 9, Username: "approver"})

	operator := contextx.GetLoginAccount(BizOperatorContext(ctx, &BizHandleParam{
		Procinst: flowentity.Procinst{CreatorId: 1, Creator: "admin"},
	}))
	require.NotNil(t, operator)
	require.EqualValues(t, 1, operator.Id, "回放身份必须是发起人，不是审批人")
	require.Equal(t, "admin", operator.Username)
	require.Equal(t, "trace-1", ctx.Value(traceKey{}), "调用方上下文里的链路信息不能被丢掉")
}
