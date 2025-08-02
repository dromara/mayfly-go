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
	SqlScriptRunFailMsg:    "数据库【<db-info id={{.dbId}}>{{.dbName}}</db-info>】的脚本 {{.filename}} 执行失败，错误：<error-text>{{.error}}</error-text>",
	SqlScriptRunSuccessMsg: "数据库【<db-info id={{.dbId}}>{{.dbName}}</db-info>】的脚本 {{.filename}} 执行成功，耗时 {{.cost}}",

	FlowUserTaskTodoMsg: "【{{.creator}}】提交的流程【{{.procdefName}}】已进入【{{.taskName}}】节点，请及时处理，去处理  <a href='#/flow/procinst-tasks'> >>></a>",
}
