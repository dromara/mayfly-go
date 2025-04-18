package imsg

import (
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	LogMsgChannelSave = iota + consts.ImsgNumMsg
	LogMsgChannelDelete

	LogMsgTmplSave
	LogMsgTmplDelete
	LogMsgTmplSend
)
