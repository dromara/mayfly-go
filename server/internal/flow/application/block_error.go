package application

import (
	"context"
	"errors"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
)

// 本文件是「资源操作被触发策略拦下」的统一出口：要不要阻断、以什么错误码出去、提示里带不带命中原因，
// 都在这里决定一次，业务模块只提供自己的话术消息 id。
//
// CodeNeedWorkTicket 是「需提交工单审批才能执行」的错误码。
//
// 给一个可判定的错误码而不是让前端去匹配提示文案：被拦下的操作要在拦截处渲染
// 「提交工单」入口，而「已被管理员禁止执行」绝不该出现这个入口。两者文案同为红色提示，
// 靠字符串区分会在换语言或改文案时静默失效（要么该给入口的地方没有，要么不该给的地方给了）。
// 取值在 int16 范围内，且与既有码（200/400/405/500/501/502）都不冲突
const CodeNeedWorkTicket int16 = 4001

// CodeNeedWarnAck 是「命中仅提醒、需操作者确认」的错误码。
//
// 与需提单码分开是必须的：两者在前端的后续动作完全不同（一个重试、一个去提单页）
const CodeNeedWarnAck int16 = 4002

// BlockSpeech 一个资源入口给操作者的话术与能力声明。
type BlockSpeech struct {
	// Forbidden 禁止执行时的话术消息 id
	Forbidden i18n.MsgId
	// Approval 需审批时的话术消息 id；为 0 表示该场景没有审批通道（如机器命令）
	Approval i18n.MsgId
	// Warn 「仅提醒」需要确认时的话术；为 0 用默认的「可直接执行或转审批」措辞。
	// 入口没有提单通道时必须换措辞，否则会指挥用户去找一个不存在的按钮
	Warn i18n.MsgId
	// RequireWarnAck 「仅提醒」是否需要操作者先确认。
	//
	// 只有能弹出「直接执行 / 转审批」确认的入口才置 true：提醒命中时操作还没执行，
	// 此刻转审批才是「只执行一次」；等执行完再提单就会跑第二遍。
	// 批量执行、key 面板这类无法逐条追问的入口置 false，保持「不阻断、只回显」
	RequireWarnAck bool
}

// NewBlockError 把处置结论翻译成给操作者的业务错误；无需阻断也无需确认时返回 nil。
//
// 「仅提醒」在 RequireWarnAck 时返回带 WarnAckCode 的错误：前端据此弹确认，
// 选「直接执行」就带 ack 重试一次，选「转审批」就走已有的一键提单（本次不执行）。
//
// 三个资源入口以前各自写了一遍「禁止→报错、需审批→带码报错」的分支，各说各话：
// 数据库与 Redis 被禁止执行时只给一句「已被禁止」，操作者看不出踩了哪条规则，而机器终端却带了原因。
// 集中到这里后，新资源类型只要复用本函数就能拿到同一套语义（含分流错误码与命中原因从句）。
// Approval 传 0 表示该场景没有审批通道（如机器命令）：此时出现「需审批」结论属于无法落地的配置，
// 按禁止执行拒而不是放行
func NewBlockError(ctx context.Context, decision *trigger.Decision, speech BlockSpeech, warnAcknowledged bool) error {
	if decision == nil {
		return nil
	}
	// 命中原因要遍历 findings 并逐条 i18n 渲染，而每个待执行语句/命令都要过这里：
	// 只在真需要拼提示时才算，不能让放行路径白付这份开销
	blockReason := func() string {
		return decision.Reason(ctx)
	}
	switch {
	case decision.RequiresApproval():
		if speech.Approval == 0 {
			// 无审批通道却配出「需审批」属配置异常，按禁止执行拒，绝不静默放行
			return errorx.NewBiz(blockMessage(ctx, speech.Forbidden, blockReason()))
		}
		return newApprovalError(ctx, speech.Approval, blockReason())
	case decision.IsForbidden():
		return errorx.NewBiz(blockMessage(ctx, speech.Forbidden, blockReason()))
	case speech.RequireWarnAck && !warnAcknowledged && len(decision.Notices()) > 0:
		return newWarnAckError(ctx, speech.Warn, blockReason())
	default:
		return nil
	}
}

// IsWarnAckError 判断错误是否为「需要操作者确认提醒」
func IsWarnAckError(err error) bool {
	var bizErr *errorx.BizError
	return errors.As(err, &bizErr) && bizErr.Code() == CodeNeedWarnAck
}

// newWarnAckError 生成带确认码的业务错误。
//
// 这里刻意把 blockMessage 的结果交给 NewBizCode 而不是 NewBizI：消息已经按请求语言渲染完，
// 再包一层会把确认码丢掉（错误码就是前端分流的唯一依据）
func newWarnAckError(ctx context.Context, msgId i18n.MsgId, reason string) *errorx.BizError {
	if msgId == 0 {
		msgId = imsg.ErrNeedWarnConfirm
	}
	return errorx.NewBizCode(CodeNeedWarnAck, blockMessage(ctx, msgId, reason))
}

// newApprovalError 生成带「需提单」错误码的业务错误，消息按请求语言渲染
func newApprovalError(ctx context.Context, msgId i18n.MsgId, reason string) *errorx.BizError {
	return errorx.NewBizCode(CodeNeedWorkTicket, blockMessage(ctx, msgId, reason))
}

// blockMessage 渲染阻断提示，并在命中原因非空时追加统一的原因从句。
//
// 原因可能为空（如「策略不可解析」这类 fail-closed 结论只有定位没有文案），
// 此时不能给操作者留一个空括号；把括号只写在这一处，就不会出现某个入口漏带原因
func blockMessage(ctx context.Context, msgId i18n.MsgId, reason string) string {
	message := i18n.TC(ctx, msgId)
	if reason == "" {
		return message
	}
	return message + i18n.TC(ctx, imsg.ErrBlockReasonSuffix, "reason", reason)
}

// WarnNotice 给无法弹确认的入口（机器终端）生成「仅提醒」命中的回显文本；无提醒时返回空串。
//
// 机器命令没有审批通道，提醒不能转成工单，但也不能只进服务端日志：
// 管理员配了「仅提醒」而操作者永远看不到，这一级别等于没配
func WarnNotice(ctx context.Context, decision *trigger.Decision) string {
	notices := decision.Notices()
	if len(notices) == 0 {
		return ""
	}
	return i18n.TC(ctx, imsg.TriggerWarnEcho, "reason", decision.Reason(ctx))
}

// IsNeedApprovalError 判断错误是否为「需提交工单审批」
func IsNeedApprovalError(err error) bool {
	var bizErr *errorx.BizError
	return errors.As(err, &bizErr) && bizErr.Code() == CodeNeedWorkTicket
}

// PreserveDecisionCode 让「需提单」「需确认」这类靠错误码分流的拦截带着原码中断请求。
//
// api 层惯例是用 biz.ErrIsNil(err) 断言失败，但它会把任何错误重新包成 code=400：
// 码一丢，前端就分不清该在拦截处给「提交工单」入口还是弹确认框，界面照常显示一句提示、
// 旁边的按钮再也不出现——这种失效从页面上完全看不出来。所以带分流码的错误必须原样抛出
func PreserveDecisionCode(err error) {
	if IsNeedApprovalError(err) || IsWarnAckError(err) {
		panic(err)
	}
}
