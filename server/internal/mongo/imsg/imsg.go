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
	LogMongoSave = iota + consts.ImsgNumMongo
	LogMongoDelete
	LogMongoRunCmd
	LogUpdateDocs
	LogDelDocs
	LogInsertDocs
	LogMongoQueryDocs
	LogMongoCreateCollection
	LogMongoDropCollection
	LogMongoDropDatabase
	LogMongoCreateIndex
	LogMongoDropIndex
	LogMongoBatchUpdate
	LogMongoBatchDelete
	LogMongoExport
	LogMongoAggregate

	ErrMongoInfoExist
	ErrMongoNotFound
	ErrMongoInvalidId
	ErrMongoDatabaseEmpty
	ErrMongoCollectionEmpty
	ErrMongoUriRequired
	ErrMongoConnFailed
	ErrMongoDocsEmpty
	ErrMongoUpdateOperatorForbidden
	ErrMongoJsonInvalid
	ErrMongoCommandEmpty
	ErrMongoCmdPermDenied
	ErrMongoIdTokenInvalid
	ErrMongoIdTokenMissing
	ErrMongoConfirmMismatch
	ErrMongoExecFailed
	ErrMongoExecTimeout
	ErrMongoDocNotMatched
	ErrMongoDocConflict
	ErrMongoUnauthorized
	ErrMongoPipelineInvalid
	ErrMongoIndexSpecsInvalid
	ErrMongoBatchCountMismatch
	ErrMongoFormatUnsupported

	// 命令语义级别的可读名称，用于拼进权限拒绝提示
	LevelReadName
	LevelDataSaveName
	LevelDataDelName
	LevelStructSaveName
	LevelStructDelName
	LevelAdminName
)
