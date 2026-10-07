package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogDbInstSave:   "DB - Save Instance",
	LogDbInstDelete: "DB - Delete Instance",

	ErrDbInstExist: "The database instance already exists",

	// db
	LogDbSave:       "DB - Save DB",
	LogDbDelete:     "DB - Delete DB",
	LogDbRunSQL:     "DB - Run SQL",
	LogDbRunSQLFile: "DB - Run SQL File",
	LogDbImportData: "DB - Import Data File",
	LogDbDump:       "DB - Export DB",

	SQLScripRunProgress: "sql execution progress",
	ErrDbNameExist:      "The database name already exists in this instance",
	ErrDbNotAccess:      "The operation permissions of database [{{.dbName}}] are not configured",

	ErrExistRunFailSQL:      "There is an execution error in sql",
	ErrNeedSubmitWorkTicket: "this operation needs approval via a work ticket, and is executed automatically once approved",

	TriggerReasonDmlRequiresApproval: "the statement type is one of the selected types",
	TriggerReasonDmlWithoutWhere:     "the update or delete has no WHERE clause",
	TriggerReasonDestructiveDdl:      "it contains destructive DDL",
	TriggerReasonSqlSize:             "the SQL size exceeds the limit",
	ErrSQLExecCancelled:              "SQL execution cancelled",
	ErrSQLSplitUnterminated:          "unterminated {{.kind}} region at line {{.line}}, the statement boundary cannot be determined",

	// db transfer
	LogDtsSave:         "dts - Save data transfer task",
	LogDtsDelete:       "dts - Delete data transfer task",
	LogDtsChangeStatus: "dts - Change status",
	LogDtsRun:          "dts - Run data transfer task",
	LogDtsStop:         "dts - Stop data transfer task",
	LogDtsDeleteFile:   "dts - Delete transfer file",
	LogDtsRunSQLFile:   "dts - Run SQL File",
	LogDtsVerify:       "dts - Verify data",

	// data sync
	LogDataSyncSave:             "datasync - Save data sync task",
	LogDataSyncDelete:           "datasync - Delete data sync task",
	LogDataSyncChangeStatus:     "datasync - Change status",
	DataSyncSuccessMsg:          "the synchronous task was executed successfully. New data: {{.count}}",
	DataSyncFailMsg:             "execution failure: {{.msg}}",
	DataSyncingMsg:              "during the execution of this task, {{.count}} has been synchronized",
	DataSyncValidationMsg:       "data validation: source={{.srcCount}}, target={{.targetCount}}, diff={{.diff}}",
	DataSyncBiDirReverseCreated: "bidirectional sync: reverse task [{{.reverseTaskId}}] created for forward task [{{.forwardTaskId}}]",

	// db mask
	LogDbMaskRuleSave:        "mask - Save masking rule",
	LogDbMaskRuleDelete:      "mask - Delete masking rule",
	LogDbMaskTagSave:         "mask - Save masking column tag",
	LogDbMaskTagDelete:       "mask - Delete masking column tag",
	ErrMaskTagNeedAlgoOrRule: "A bind-action column tag requires an algorithm or a related rule",

	// scheduled task
	ErrTaskCronInvalid: "The cron expression [{{.cron}}] of the scheduled task is invalid, so it will never run as planned: {{.reason}}",
}
