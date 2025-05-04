package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogDbInstSave:   "DB-保存数据库实例",
	LogDbInstDelete: "DB-删除数据库实例",

	ErrDbInstExist: "该数据库实例已存在",

	// db
	LogDbSave:   "DB-保存数据库",
	LogDbDelete: "DB-删除数据库",
	LogDbRunSql: "DB-运行SQL",
	LogDbDump:   "DB-导出数据库",

	SqlScriptRunFail:    "sql脚本执行失败",
	SqlScriptRunSuccess: "sql脚本执行成功",
	SqlScripRunProgress: "sql执行进度",
	DbDumpErr:           "数据库导出失败",
	ErrDbNameExist:      "该实例下数据库名已存在",
	ErrDbNotAccess:      "未配置数据库【{{.dbName}}】的操作权限",

	ErrExistRunFailSql:      "存在执行错误的sql",
	ErrNeedSubmitWorkTicket: "该操作需要提交工单审批执行",

	// db transfer
	LogDtsSave:         "dts-保存数据迁移任务",
	LogDtsDelete:       "dts-删除数据迁移任务",
	LogDtsChangeStatus: "dts-启停任务",
	LogDtsRun:          "dts-执行数据迁移任务",
	LogDtsStop:         "dts-终止数据迁移任务",
	LogDtsDeleteFile:   "dts-删除迁移文件",
	LogDtsRunSqlFile:   "dts-执行sql文件",

	// data sync
	LogDataSyncSave:         "datasync-保存数据同步任务",
	LogDataSyncDelete:       "datasync-删除数据同步任务",
	LogDataSyncChangeStatus: "datasync-启停任务",
}
