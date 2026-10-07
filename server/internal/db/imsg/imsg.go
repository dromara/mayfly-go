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
)
