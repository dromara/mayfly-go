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
	LogDbRunSql
	LogDbRunSqlFile
	LogDbDump

	SqlScripRunProgress
	ErrDbNameExist
	ErrDbNotAccess

	ErrExistRunFailSql
	ErrNeedSubmitWorkTicket
	ErrSqlExecCancelled

	// db transfer
	LogDtsSave
	LogDtsDelete
	LogDtsChangeStatus
	LogDtsRun
	LogDtsStop
	LogDtsDeleteFile
	LogDtsRunSqlFile
	LogDtsVerify

	// data sync
	LogDataSyncSave
	LogDataSyncDelete
	LogDataSyncChangeStatus
	DataSyncSuccessMsg
	DataSyncFailMsg
	DataSyncingMsg

	// db mask
	LogDbMaskRuleSave
	LogDbMaskRuleDelete
	LogDbMaskTagSave
	LogDbMaskTagDelete
	ErrMaskTagNeedAlgoOrRule
)
