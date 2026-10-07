package imsg

import (
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	LogRedisSave = iota + consts.ImsgNumRedis
	LogRedisDelete
	LogRedisRunCmd
	LogRedisKeyOp

	ErrRedisInfoExist
	ErrSubmitFlowRunCmd

	// ErrRunCmdFromConsole key 面板的类型化操作命中「需审批」时使用：这里没有提单表单，
	// 得告诉操作者去哪儿提这张单，而不是让他对着一句「请提交工单」找按钮
	ErrRunCmdFromConsole

	// 触发策略检查项命中原因（用于拦截提示）
	TriggerReasonWriteCmd
	TriggerReasonDangerousCmd
	TriggerReasonCmdIn
	ErrHasRunFailCmd
	ErrRedisKeyTypeUnsupported
	ErrRedisKeyViewUnsupported
	ErrRedisKeyNotFound
	ErrRedisDangerousCmd
	ErrRedisKeyAlreadyExist
	ErrRedisCopySkipped
)
