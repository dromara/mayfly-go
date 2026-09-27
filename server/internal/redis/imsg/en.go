package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogRedisSave:   "Redis - Save",
	LogRedisDelete: "Redis - Delete",
	LogRedisRunCmd: "Redis - Run Cmd",
	LogRedisKeyOp:  "Redis - Key Data Operation",

	ErrRedisInfoExist:          "The Redis information already exists",
	ErrSubmitFlowRunCmd:        "This operation needs to submit a work ticket for approval",
	ErrHasRunFailCmd:           "A command failed to execute",
	ErrRedisKeyTypeUnsupported: "The built-in view of data type [{{.type}}] is unavailable, please use the command console",
	ErrRedisKeyViewUnsupported: "The view [{{.view}}] does not support the data type [{{.type}}]",
	ErrRedisKeyNotFound:        "The key no longer exists, please refresh and try again",
	ErrRedisDangerousCmd:       "This is a dangerous command which requires the data delete permission",
	ErrRedisKeyAlreadyExist:    "The target key [{{.key}}] already exists, renaming would overwrite it silently, please use another name",
	ErrRedisCopySkipped:        "Copy not performed: the target key [{{.key}}] already exists without replace, or the source key is gone",
}
