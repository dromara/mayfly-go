package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogRedisSave:   "Redis-保存",
	LogRedisDelete: "Redis-删除",
	LogRedisRunCmd: "Redis-执行命令",

	ErrRedisInfoExist:   "该Redis信息已存在",
	ErrSubmitFlowRunCmd: "该操作需要提交工单审批执行",
	ErrHasRunFailCmd:    "存在执行失败的命令",
}
