package imsg

import "mayfly-go/pkg/i18n"

// En Mongo 模块英文文案
var En = map[i18n.MsgId]string{
	LogMongoSave:             "Mongo - Save Instance",
	LogMongoDelete:           "Mongo - Delete Instance",
	LogMongoRunCmd:           "Mongo - Run Command",
	LogUpdateDocs:            "Mongo - Update Documents",
	LogDelDocs:               "Mongo - Delete Documents",
	LogInsertDocs:            "Mongo - Insert Documents",
	LogMongoQueryDocs:        "Mongo - Query Documents",
	LogMongoCreateCollection: "Mongo - Create Collection",
	LogMongoDropCollection:   "Mongo - Drop Collection",
	LogMongoDropDatabase:     "Mongo - Drop Database",

	ErrMongoInfoExist:               "that connection information already exists",
	ErrMongoNotFound:                "mongo instance not found",
	ErrMongoInvalidId:               "invalid mongo instance id",
	ErrMongoDatabaseEmpty:           "database name cannot be empty",
	ErrMongoCollectionEmpty:         "collection name cannot be empty",
	ErrMongoUriRequired:             "connection uri cannot be empty",
	ErrMongoConnFailed:              "mongo connection failed: {{.detail}}",
	ErrMongoDocsEmpty:               "document content cannot be empty",
	ErrMongoUpdateOperatorForbidden: "document field names must not start with $; the update instruction is derived from the submitted content",
	ErrMongoJsonInvalid:             "{{.field}} cannot be parsed as a json document: {{.detail}}. note: json keys must be double-quoted",
	ErrMongoCommandEmpty:            "command content cannot be empty",
	ErrMongoCmdPermDenied:           "command {{.command}} is a {{.level}} operation and requires permission {{.permission}}",
	ErrMongoIdTokenInvalid:          "the document id token is invalid or has been tampered with",
	ErrMongoIdTokenMissing:          "the document has no _id field (it may have been excluded by the projection), so it cannot be located",
	ErrMongoConfirmMismatch:         "confirmation name mismatched, type {{.expect}} to continue",
	ErrMongoExecFailed:              "mongo operation failed: {{.detail}}",
	ErrMongoExecTimeout:             "mongo operation was cancelled after {{.limit}}s, narrow the query or add an index for the involved fields",
	ErrMongoDocNotMatched:           "no document matched, the id may have changed or the document was already deleted",
	ErrMongoDocConflict:             "this document was modified by someone else, the write was cancelled; refresh and edit again",
	ErrMongoUnauthorized:            "that instance requires authentication: the uri carries no credentials, or the account lacks this permission ({{.detail}}). Edit the instance and provide user/password with authSource",
	ErrMongoPipelineInvalid:         "the aggregation pipeline must be a non-empty array of stages, e.g. [{\"$match\":{...}}]",
	ErrMongoIndexSpecsInvalid:       "index specifications must be a non-empty array where each item has a key field, e.g. [{\"key\":{\"a\":1},\"name\":\"a_1\"}]",
	ErrMongoBatchCountMismatch:      "the matched count changed: expected {{.expect}} but got {{.actual}}; nothing was executed, please confirm again",
	ErrMongoFormatUnsupported:       "unsupported export format {{.format}}, currently supported: {{.supported}}",

	LevelReadName:       "read-only",
	LevelDataSaveName:   "data-writing",
	LevelDataDelName:    "data-deleting",
	LevelStructSaveName: "schema-changing",
	LevelStructDelName:  "schema-dropping",
	LevelAdminName:      "server-administering",
}
