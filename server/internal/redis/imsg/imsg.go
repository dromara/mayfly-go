package imsg

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	LogRedisSave = iota + consts.ImsgNumRedis
	LogRedisDelete
	LogRedisRunCmd

	ErrRedisInfoExist
	ErrSubmitFlowRunCmd
	ErrHasRunFailCmd
)
