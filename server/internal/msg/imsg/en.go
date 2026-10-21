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
	DbTaskRunFailMsg:       "database task [{{.taskName}}] failed, error: <error-text>{{.error}}</error-text>",
	DbSchemaDriftMsg:       "database [<db-info id={{.dbId}}>{{.dbName}}</db-info>] has {{.count}} table(s) whose structure no longer matches the baseline: {{.detail}}",
	SQLScriptRunFailMsg:    "Script {{.filename}} execution failed on database [<db-info id={{.dbId}}>{{.dbName}}</db-info>], error: <error-text>{{.error}}</error-text>",
	SQLScriptRunSuccessMsg: "Script {{.filename}} executed successfully on database [<db-info id={{.dbId}}>{{.dbName}}</db-info>], cost {{.cost}}",

	FlowUserTaskTodoMsg: "Work order [{{.procdefName}}] submitted by [{{.creator}}] is now at [{{.taskName}}] node. Please process it promptly. <a href='#/flow/procinst-tasks'>Handle it >>></a>",

	ErrMsgTmplNotExist:  "message template [{{.code}}] does not exist",
	ErrMsgTmplDisabled:  "message template [{{.code}}] is disabled and cannot be used for notifications",
	ErrMsgTmplNoChannel: "message template [{{.code}}] is not associated with any channel, so notifications cannot be delivered; associate a channel with it first",

	ErrMsgTmplIdNotExist:      "message template [{{.id}}] does not exist",
	ErrMsgTmplBizIdRequired:   "the business id is required",
	ErrMsgTmplBizTypeRequired: "the business type is required",
	ErrMsgTmplBizNotBound:     "no message template is associated with this business",
}
