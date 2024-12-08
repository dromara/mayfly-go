package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogRedisSave:   "Redis - Save",
	LogRedisDelete: "Redis - Delete",
	LogRedisRunCmd: "Redis - Run Cmd",

	ErrRedisInfoExist:   "The Redis information already exists",
	ErrSubmitFlowRunCmd: "This operation needs to submit a work ticket for approval",
	ErrHasRunFailCmd:    "A command failed to execute",
}
