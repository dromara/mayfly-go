package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogDbInstSave:   "DB - Save Instance",
	LogDbInstDelete: "DB - Delete Instance",

	ErrDbInstExist: "The database instance already exists",

	// db
	LogDbSave:   "DB - Save DB",
	LogDbDelete: "DB - Delete DB",
	LogDbRunSql: "DB - Run SQL",
	LogDbDump:   "DB - Export DB",

	SqlScriptRunFail:    "sql script failed to execute",
	SqlScriptRunSuccess: "sql script executed successfully",
	SqlScripRunProgress: "sql execution progress",
	DbDumpErr:           "Database export failed",
	ErrDbNameExist:      "The database name already exists in this instance",
	ErrDbNotAccess:      "The operation permissions of database [{{.dbName}}] are not configured",

	ErrExistRunFailSql:      "There is an execution error in sql",
	ErrNeedSubmitWorkTicket: "This operation needs to submit a work ticket for approval",

	// db transfer
	LogDtsSave:         "dts - Save data transfer task",
	LogDtsDelete:       "dts - Delete data transfer task",
	LogDtsChangeStatus: "dts - Change status",
	LogDtsRun:          "dts - Run data transfer task",
	LogDtsStop:         "dts - Stop data transfer task",
	LogDtsDeleteFile:   "dts - Delete transfer file",
	LogDtsRunSqlFile:   "dts - Run SQL File",

	// data sync
	LogDataSyncSave:         "datasync - Save data sync task",
	LogDataSyncDelete:       "datasync - Delete data sync task",
	LogDataSyncChangeStatus: "datasync - Change status",
}
