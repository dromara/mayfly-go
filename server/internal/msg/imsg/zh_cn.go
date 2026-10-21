package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogMsgChannelSave:   "消息渠道-保存",
	LogMsgChannelDelete: "消息渠道-删除",

	LogMsgTmplSave:   "消息模板-保存",
	LogMsgTmplDelete: "消息模板-删除",
	LogMsgTmplSend:   "消息模板-发送",

	LoginMsg: "于[{{.ip}}]-[{{.time}}]登录",

	MachineFileUploadSuccessMsg: "[{{.filename}}] -> <machine-info code={{.machineCode}}>{{.machineName}}</machine-info>【{{.path}}】",
	MachineFileUploadFailMsg:    "[{{.filename}}] -> <machine-info code={{.machineCode}}>{{.machineName}}</machine-info>【{{.path}}】。错误信息：<error-text>{{.error}}</error-text>",

	DbDumpFailMsg:          "数据库【<db-info id={{.dbId}}>{{.dbName}}</db-info>】导出失败，错误信息：<error-text>{{.error}}</error-text>",
	DbTaskRunFailMsg:       "数据库任务【{{.taskName}}】执行失败，错误信息：<error-text>{{.error}}</error-text>",
	DbSchemaDriftMsg:       "数据库【<db-info id={{.dbId}}>{{.dbName}}</db-info>】有 {{.count}} 张表的结构与基线不一致：{{.detail}}",
	SQLScriptRunFailMsg:    "数据库【<db-info id={{.dbId}}>{{.dbName}}</db-info>】的脚本 {{.filename}} 执行失败，错误：<error-text>{{.error}}</error-text>",
	SQLScriptRunSuccessMsg: "数据库【<db-info id={{.dbId}}>{{.dbName}}</db-info>】的脚本 {{.filename}} 执行成功，耗时 {{.cost}}",

	FlowUserTaskTodoMsg: "【{{.creator}}】提交的流程【{{.procdefName}}】已进入【{{.taskName}}】节点，请及时处理，去处理  <a href='#/flow/procinst-tasks'> >>></a>",

	ErrMsgTmplNotExist:  "消息模板[{{.code}}]不存在",
	ErrMsgTmplDisabled:  "消息模板[{{.code}}]已停用，不能用于发送通知",
	ErrMsgTmplNoChannel: "消息模板[{{.code}}]未关联任何消息渠道，通知发不出去；请先为该模板关联渠道",

	ErrMsgTmplIdNotExist:      "消息模板[{{.id}}]不存在",
	ErrMsgTmplBizIdRequired:   "业务 ID 不能为空",
	ErrMsgTmplBizTypeRequired: "业务类型不能为空",
	ErrMsgTmplBizNotBound:     "该业务尚未关联消息模板",
}
