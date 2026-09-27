package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogRedisSave:   "Redis-保存",
	LogRedisDelete: "Redis-删除",
	LogRedisRunCmd: "Redis-执行命令",
	LogRedisKeyOp:  "Redis-数据操作",

	ErrRedisInfoExist:          "该Redis信息已存在",
	ErrSubmitFlowRunCmd:        "该操作需要提交工单审批执行",
	ErrHasRunFailCmd:           "存在执行失败的命令",
	ErrRedisKeyTypeUnsupported: "数据类型[{{.type}}]暂无内置视图，请使用命令控制台操作",
	ErrRedisKeyViewUnsupported: "视图[{{.view}}]不支持数据类型[{{.type}}]",
	ErrRedisKeyNotFound:        "该key已不存在，请刷新后重试",
	ErrRedisDangerousCmd:       "该命令为高危命令，需要数据删除权限",
	ErrRedisKeyAlreadyExist:    "目标 key[{{.key}}] 已存在，重命名会直接覆盖它，请换一个名字",
	ErrRedisCopySkipped:        "复制未执行：目标 key[{{.key}}] 已存在且未勾选覆盖，或源 key 已不存在",
}
