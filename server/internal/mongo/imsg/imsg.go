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
	LogMongoSave = iota + consts.ImsgNumMongo
	LogMongoDelete
	LogMongoRunCmd
	LogUpdateDocs
	LogDelDocs
	LogInsertDocs

	ErrMongoInfoExist
)
