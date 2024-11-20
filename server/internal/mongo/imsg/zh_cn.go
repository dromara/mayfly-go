package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogMongoSave:   "Mongo-保存",
	LogMongoDelete: "Mongo-删除",
	LogMongoRunCmd: "Mongo-执行命令",
	LogUpdateDocs:  "Mongo-更新文档",
	LogDelDocs:     "Mongo-删除文档",
	LogInsertDocs:  "Mongo-插入文档",

	ErrMongoInfoExist: "该信息已存在",
}
