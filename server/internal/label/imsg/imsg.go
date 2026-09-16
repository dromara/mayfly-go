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
	LogLabelSave = iota + consts.ImsgNumLabel
	LogLabelDelete

	// ---- 业务校验错误 ----
	ErrLabelKeyRequired
	ErrLabelValueRequired
	ErrLabelDuplicate
	ErrLabelInUse
	ErrLabelKeyValueChanged
)
