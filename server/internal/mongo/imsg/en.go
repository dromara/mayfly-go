package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogMongoSave:   "Mongo - Save",
	LogMongoDelete: "Mongo-Delete",
	LogMongoRunCmd: "Mongo - Run Cmd",
	LogUpdateDocs:  "Mongo - Update Documents",
	LogDelDocs:     "Mongo - Delete Documents",
	LogInsertDocs:  "Mongo - Insert Documents",

	ErrMongoInfoExist: "that information already exists",
}
