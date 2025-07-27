package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogMsgChannelSave:   "Message channel- save",
	LogMsgChannelDelete: "Message channel- delete",

	LogMsgTmplSave:   "Message template- save",
	LogMsgTmplDelete: "Message template- delete",
	LogMsgTmplSend:   "Message template- send",

	LoginMsg: "Log in to [{{.ip}}]-[{{.time}}]",

	MachineFileUploadSuccessMsg: "[{{.filename}}] -> {{.machineName}}[{{.machineIp}}:{{.path}}]",
	MachineFileUploadFailMsg:    "[{{.filename}}] -> {{.machineName}}[{{.machineIp}}:{{.path}}]. error: {{.error}}",

	DbDumpFailMsg:          "Database dump failed, error: {{.error}}",
	SqlScriptRunFailMsg:    "Script {{.filename}} execution failed on database {{.db}}, error: {{.error}}",
	SqlScriptRunSuccessMsg: "Script {{.filename}} executed successfully on database {{.db}}, cost {{.cost}}",
}
