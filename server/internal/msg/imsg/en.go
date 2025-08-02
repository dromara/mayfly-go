package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogMsgChannelSave:   "Message channel- save",
	LogMsgChannelDelete: "Message channel- delete",

	LogMsgTmplSave:   "Message template- save",
	LogMsgTmplDelete: "Message template- delete",
	LogMsgTmplSend:   "Message template- send",

	LoginMsg: "Log in to [{{.ip}}]-[{{.time}}]",

	MachineFileUploadSuccessMsg: "[{{.filename}}] -> <machine-info code={{.machineCode}}>{{.machineName}}</machine-info> [{{.path}}]",
	MachineFileUploadFailMsg:    "[{{.filename}}] -> <machine-info code={{.machineCode}}>{{.machineName}}</machine-info> [{{.path}}]. error: {{.error}}",

	DbDumpFailMsg:          "Database [<db-info id={{.dbId}}>{{.dbName}}</db-info>] dump failed, error: <error-text>{{.error}}</error-text>",
	SqlScriptRunFailMsg:    "Script {{.filename}} execution failed on database [<db-info id={{.dbId}}>{{.dbName}}</db-info>], error: <error-text>{{.error}}</error-text>",
	SqlScriptRunSuccessMsg: "Script {{.filename}} executed successfully on database [<db-info id={{.dbId}}>{{.dbName}}</db-info>], cost {{.cost}}",

	FlowUserTaskTodoMsg: "Work order [{{.procdefName}}] submitted by [{{.creator}}] is now at [{{.taskName}}] node. Please process it promptly. <a href='#/flow/procinst-tasks'>Handle it >>></a>",
}
