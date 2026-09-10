package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogDbInstSave:   "DB - Save Instance",
	LogDbInstDelete: "DB - Delete Instance",

	ErrDbInstExist: "The database instance already exists",

	// db
	LogDbSave:       "DB - Save DB",
	LogDbDelete:     "DB - Delete DB",
	LogDbRunSql:     "DB - Run SQL",
	LogDbRunSqlFile: "DB - Run SQL File",
	LogDbDump:       "DB - Export DB",

	SqlScripRunProgress: "sql execution progress",
	ErrDbNameExist:      "The database name already exists in this instance",
	ErrDbNotAccess:      "The operation permissions of database [{{.dbName}}] are not configured",

	ErrExistRunFailSql:      "There is an execution error in sql",
	ErrNeedSubmitWorkTicket: "This operation needs to submit a work ticket for approval",
	ErrSqlExecCancelled:     "SQL execution cancelled",
	ErrSqlSplitUnterminated: "unterminated {{.kind}} region at line {{.line}}, the statement boundary cannot be determined",

	// db transfer
	LogDtsSave:         "dts - Save data transfer task",
	LogDtsDelete:       "dts - Delete data transfer task",
	LogDtsChangeStatus: "dts - Change status",
	LogDtsRun:          "dts - Run data transfer task",
	LogDtsStop:         "dts - Stop data transfer task",
	LogDtsDeleteFile:   "dts - Delete transfer file",
	LogDtsRunSqlFile:   "dts - Run SQL File",
	LogDtsVerify:       "dts - Verify data",

	// data sync
	LogDataSyncSave:         "datasync - Save data sync task",
	LogDataSyncDelete:       "datasync - Delete data sync task",
	LogDataSyncChangeStatus: "datasync - Change status",
	DataSyncSuccessMsg:      "the synchronous task was executed successfully. New data: {{.count}}",
	DataSyncFailMsg:         "execution failure: {{.msg}}",
	DataSyncingMsg:          "during the execution of this task, {{.count}} has been synchronized",

	// db mask
	LogDbMaskRuleSave:        "mask - Save masking rule",
	LogDbMaskRuleDelete:      "mask - Delete masking rule",
	LogDbMaskTagSave:         "mask - Save masking column tag",
	LogDbMaskTagDelete:       "mask - Delete masking column tag",
	ErrMaskTagNeedAlgoOrRule: "A bind-action column tag requires an algorithm or a related rule",
}
