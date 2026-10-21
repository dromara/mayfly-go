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
	LogMsgChannelSave = iota + consts.ImsgNumMsg
	LogMsgChannelDelete

	LogMsgTmplSave
	LogMsgTmplDelete
	LogMsgTmplSend

	LoginMsg

	MachineFileUploadSuccessMsg
	MachineFileUploadFailMsg

	DbDumpFailMsg
	DbTaskRunFailMsg
	DbSchemaDriftMsg
	SQLScriptRunFailMsg
	SQLScriptRunSuccessMsg

	FlowUserTaskTodoMsg

	// 通知模板「可发性」校验：业务侧（触发器/任务/机器计划任务）保存结果通知配置时前置拦截，
	// 判据与 application.MsgTmpl.Send 同源。追加在末尾——iota 平移会让已写进 t_sys_log 的
	// 历史操作日志集体错译
	ErrMsgTmplNotExist
	ErrMsgTmplDisabled
	ErrMsgTmplNoChannel

	// 模板-业务关联（流程定义等按 bizId/bizType 取模板发送）的校验文案
	ErrMsgTmplIdNotExist
	ErrMsgTmplBizIdRequired
	ErrMsgTmplBizTypeRequired
	ErrMsgTmplBizNotBound
)
