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

	// transfer export file
	LogDtsDownloadFile: "dts - Download transfer export file",

	// atomic batch execution
	ErrAtomicUnsupportedStmt: "atomic execution supports only INSERT/UPDATE/DELETE, statement [{{.sql}}] is none of them: DDL is committed implicitly on some databases and cannot be rolled back, and query results take no part in rollback",
	ErrAtomicBatchRolledBack: "statement {{.index}} of {{.total}} failed, the whole batch has been rolled back (previously succeeded statements took no effect either): {{.error}}",

	ErrSqlExecNotFoundToCancel: "no such running execution to cancel (it finished a while ago, ran on another serving instance, or the serving process restarted)",
	ErrSqlExecJustFinished:     "the execution just finished on its own; there is nothing to cancel",
	ErrSqlExecIdConflict:       "the execution id collides with another user's running execution; this request is rejected, please retry",
	ErrSqlExecNotOwned:         "only executions started by yourself can be cancelled",

	TriggerReasonRowsExceed:    "the estimated affected rows reach the threshold (or this database cannot produce an estimate)",
	TriggerReasonFullScan:      "the execution plan contains a full table/index scan",
	TriggerReasonBigTableDdl:   "the target table reaches the large-table threshold, confirm locking and duration risks before the structural change",
	TriggerReasonNoPrimaryKey:  "the target table has no primary key",
	ErrReviewCheckNotEvaluated: "this check produced no verdict, which does not mean it did not match: {{.reason}}",
	ErrReviewNoSqlToAnalyze:    "no statement to analyze",
	ErrReviewNoTargetDb:        "a target database is required, otherwise the statement cannot be parsed and estimated with its dialect",
	ErrReviewNoTargetTable:     "the target table cannot be determined, so its size is unavailable",
	ErrReviewNoTableStats:      "statistics of the target table are unavailable, so its size cannot be judged",
	ReviewEstimateUnavailable:  "no row estimate is available this time: {{.reason}} (this does not mean zero rows affected, nor that the statement is safe)",

	LogDbCancelSession: "db - Cancel database session",
	LogDbCancelExec:    "db - Cancel SQL execution",

	ErrSessionCancelFailed: "failed to cancel session {{.sessionId}} (it may have already ended, or the statement finished): {{.reason}}",

	LogDbGenRollback: "db - Generate rollback SQL",

	ErrRollbackRecordNotFound:   "the sql execution record (id={{.id}}) was not found, so the old values captured at that time are unavailable",
	ErrRollbackNotExecuted:      "the sql execution record (id={{.id}}) is not a successfully executed change: the old values belong to a statement that never took effect, so rolling back would write stale values into live data",
	ErrRollbackUnsupportedType:  "only UPDATE, DELETE and INSERT execution records carry row data usable for rollback, this record is of type [{{.type}}]",
	ErrRollbackNoOldValue:       "this execution record has no old values: the capture either failed at execution time (see server logs) or the statement cannot be located row by row, so no rollback sql can be generated",
	ErrRollbackOldValueInvalid:  "the recorded row data cannot be parsed into a row set, no rollback sql was generated: {{.reason}}",
	ErrRollbackNoTable:          "the target table [{{.table}}] no longer exists or is unreadable, rollback sql cannot be shaped against it",
	ErrRollbackColumnGone:       "column [{{.column}}] from the captured row data no longer exists in table [{{.table}}]; generating anyway would misalign values with columns, so no rollback sql was generated",
	ErrRollbackPkChanged:        "the statement changed primary key column [{{.column}}]; old values hold only the previous key, so the updated rows cannot be located and no rollback sql was generated",
	RollbackTruncatedNote:       "row data is captured up to {{.cap}} rows, anything beyond was never recorded, so this covers only the captured rows",
	ErrRollbackNoPrimaryKey:     "table [{{.table}}] has no primary key, so rows cannot be located uniquely; a rollback without a key condition would affect other rows sharing the same values, so no rollback sql was generated",
	RollbackStmtUnparsedNote:    "the original statement could not be parsed; if that UPDATE also changed primary key columns, only the non-key columns are restored here (key restore needs manual handling)",
	RollbackUnverifiedNote:      "no post-execution row snapshot is available for this record (created before the snapshot feature or capture failed at that time), so no concurrency check was performed: the rollback may overwrite later modifications to these rows",
	RollbackBlockedNote:         "{{.count}} row(s) were modified, deleted or cannot be located after the change; their rollback statements were not generated and need manual reconciliation",
	ErrRollbackNoInsertSnapshot: "this INSERT record carries no captured insertion values (statement shape unsupported or capture failed at that time), so the newly inserted rows cannot be located and no rollback sql was generated",

	ErrSchemaDiffSameDb:        "a schema comparison needs two different databases: comparing a database with itself only yields 'no difference', which is not a result",
	ErrSchemaDiffNoTables:      "no table is available to compare on either side (the table list is empty, or the current account can read no tables)",
	ErrSchemaDiffTooManyTables: "at most {{.max}} tables can be compared at a time, please narrow the table list",
	SchemaDiffTruncatedNote:    "no table list was given, so only the first {{.max}} tables were compared ({{.total}} in total on both sides); the rest took part in neither verdict",

	LogDbExportQuery: "db - Export query result set",

	ErrQueryExportEmptySql:          "there is no statement to export",
	ErrQueryExportReadOnlyOnly:      "exporting re-executes the statement, so only querying statements are allowed; the current statement type is [{{.type}}]",
	ErrQueryExportFormatUnsupported: "the export format [{{.format}}] does not apply to a query result set, supported formats: {{.supported}}",

	LogDbExplainPlan: "db - Read SQL execution plan",

	ErrExplainUnavailable: "no execution plan is available this time: {{.reason}} (this does not mean the statement needs no plan, nor that it is safe)",

	ErrColumnValuesNoColumn:    "column [{{.column}}] does not exist in table [{{.table}}] (or it has been dropped), so its values were not sampled",
	ErrColumnValuesUnsupported: "this dialect provides no pagination rewriter, so no values were sampled rather than guessing a row-limit syntax",

	ErrSchemaBaselineEmpty:      "no table structure could be read from this database, so a baseline cannot be taken",
	SchemaBaselineTruncatedNote: "the baseline covers only the first {{.max}} tables ({{.total}} in this database); changes to the remaining tables are not reported as drift",
	SchemaDriftColumnCount:      "column count {{.from}}→{{.to}}",
	SchemaDriftPrimaryKeys:      "primary key {{.from}}→{{.to}}",
	SchemaDriftIndexCount:       "index count {{.from}}→{{.to}}",
	SchemaDriftColumnDefs:       "column definitions or column order changed",
	SchemaDriftNoPk:             "no primary key",
	SchemaDriftTableAdded:       "a table that the baseline does not contain",
	SchemaDriftTableRemoved:     "a table the baseline contains is no longer in the database",
	LogDbSchemaBaselineSave:     "db - Take table structure baseline",
	LogDbSchemaBaselineDel:      "db - Delete table structure baseline",
}
