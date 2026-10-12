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
	// db inst
	LogDbInstSave = iota + consts.ImsgNumDb
	LogDbInstDelete

	ErrDbInstExist

	// db
	LogDbSave
	LogDbDelete
	LogDbRunSQL
	LogDbRunSQLFile
	LogDbImportData
	LogDbDump

	SQLScripRunProgress
	ErrDbNameExist
	ErrDbNotAccess

	ErrExistRunFailSQL
	ErrNeedSubmitWorkTicket

	// 触发策略检查项命中原因（用于拦截提示）
	TriggerReasonDmlRequiresApproval
	TriggerReasonDmlWithoutWhere
	TriggerReasonDestructiveDdl
	TriggerReasonSqlSize
	ErrSQLExecCancelled
	ErrSQLSplitUnterminated

	// db transfer
	LogDtsSave
	LogDtsDelete
	LogDtsChangeStatus
	LogDtsRun
	LogDtsStop
	LogDtsDeleteFile
	LogDtsRunSQLFile
	LogDtsVerify

	// data sync
	LogDataSyncSave
	LogDataSyncDelete
	LogDataSyncChangeStatus
	DataSyncSuccessMsg
	DataSyncFailMsg
	DataSyncingMsg
	DataSyncValidationMsg
	DataSyncBiDirReverseCreated

	// db mask
	LogDbMaskRuleSave
	LogDbMaskRuleDelete
	LogDbMaskTagSave
	LogDbMaskTagDelete
	ErrMaskTagNeedAlgoOrRule

	// 定时调度（同步任务与迁移任务共用）
	ErrTaskCronInvalid

	// 迁移导出文件下载：数据类产物被谁取走必须可查，因此单独留痕。
	// 新增 id 一律追加在末尾：id 由 iota 连续生成，插入中间会重编号已写入历史日志的 id
	LogDtsDownloadFile

	ErrAtomicUnsupportedStmt
	ErrAtomicBatchRolledBack

	ErrSqlExecNotFoundToCancel
	ErrSqlExecNotOwned

	// 变更预检与库侧派生事实
	TriggerReasonRowsExceed
	TriggerReasonFullScan
	TriggerReasonBigTableDdl
	TriggerReasonNoPrimaryKey
	ErrReviewCheckNotEvaluated
	ErrReviewNoSqlToAnalyze
	ErrReviewNoTargetDb
	ErrReviewNoTargetTable
	ErrReviewNoTableStats
	ReviewEstimateUnavailable

	LogDbCancelSession
	LogDbCancelExec

	// 取消会话失败：库侧拒绝原因得包成人话再上抛，不能直接把 `Error 1094 ...` 丢给界面
	ErrSessionCancelFailed

	// 回滚 SQL 生成
	LogDbGenRollback
	ErrRollbackRecordNotFound
	ErrRollbackNotExecuted
	ErrRollbackUnsupportedType
	ErrRollbackNoOldValue
	ErrRollbackOldValueInvalid
	ErrRollbackNoTable
	ErrRollbackColumnGone
	ErrRollbackPkChanged
	RollbackTruncatedNote
	ErrRollbackNoPrimaryKey
	RollbackStmtUnparsedNote
	RollbackUnverifiedNote
	RollbackBlockedNote
	ErrRollbackNoInsertSnapshot

	// 结构对比（漂移检测）
	ErrSchemaDiffSameDb
	ErrSchemaDiffNoTables
	ErrSchemaDiffTooManyTables
	SchemaDiffTruncatedNote

	// 查询结果集全量导出
	ErrQueryExportEmptySql
	ErrQueryExportReadOnlyOnly
	ErrQueryExportFormatUnsupported
	LogDbExportQuery

	// 执行计划可视化：拿不到计划必须明说，不能给出一张空图让人以为「计划里没有扫描」
	ErrExplainUnavailable
	LogDbExplainPlan

	// 列值采样（编辑器值补全）
	ErrColumnValuesNoColumn
	ErrColumnValuesUnsupported

	// 表结构漂移基线
	ErrSchemaBaselineEmpty
	SchemaBaselineTruncatedNote
	SchemaDriftColumnCount
	SchemaDriftPrimaryKeys
	SchemaDriftIndexCount
	SchemaDriftColumnDefs
	SchemaDriftNoPk
	SchemaDriftTableAdded
	SchemaDriftTableRemoved
	LogDbSchemaBaselineSave
	LogDbSchemaBaselineDel

	// 取消比结算晚到一步（如长语句尾点取消）：与「查无此执行」是两种事实，必须分开回答
	ErrSqlExecJustFinished

	// 跨属主 execId 碰撞：登记的顶替行为不得绕过取消侧的属主校验
	ErrSqlExecIdConflict
)
