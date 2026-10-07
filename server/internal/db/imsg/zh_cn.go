package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogDbInstSave:   "DB-保存数据库实例",
	LogDbInstDelete: "DB-删除数据库实例",

	ErrDbInstExist: "该数据库实例已存在",

	// db
	LogDbSave:       "DB-保存数据库",
	LogDbDelete:     "DB-删除数据库",
	LogDbRunSQL:     "DB-运行SQL",
	LogDbRunSQLFile: "DB-执行SQL文件",
	LogDbImportData: "DB-导入数据文件",
	LogDbDump:       "DB-导出数据库",

	SQLScripRunProgress: "sql执行进度",
	ErrDbNameExist:      "该实例下数据库名已存在",
	ErrDbNotAccess:      "未配置数据库【{{.dbName}}】的操作权限",

	ErrExistRunFailSQL:      "存在执行错误的sql",
	ErrNeedSubmitWorkTicket: "该操作需要提交工单审批执行，审批通过后会自动执行本次操作",

	TriggerReasonDmlRequiresApproval: "语句类型属于所选类型",
	TriggerReasonDmlWithoutWhere:     "更新或删除缺少 WHERE 条件",
	TriggerReasonDestructiveDdl:      "包含破坏性 DDL 操作",
	TriggerReasonSqlSize:             "SQL 体积超过上限",
	ErrSQLExecCancelled:              "SQL执行已取消",
	ErrSQLSplitUnterminated:          "SQL第{{.line}}行存在未闭合的 {{.kind}} 区域，无法判定语句边界，请补全后重试",

	// db transfer
	LogDtsSave:         "dts-保存数据迁移任务",
	LogDtsDelete:       "dts-删除数据迁移任务",
	LogDtsChangeStatus: "dts-启停任务",
	LogDtsRun:          "dts-执行数据迁移任务",
	LogDtsStop:         "dts-终止数据迁移任务",
	LogDtsDeleteFile:   "dts-删除迁移文件",
	LogDtsRunSQLFile:   "dts-执行sql文件",
	LogDtsVerify:       "dts-数据校验",

	// data sync
	LogDataSyncSave:             "datasync-保存数据同步任务",
	LogDataSyncDelete:           "datasync-删除数据同步任务",
	LogDataSyncChangeStatus:     "datasync-启停任务",
	DataSyncSuccessMsg:          "执行成功，本次同步{{.count}}条",
	DataSyncFailMsg:             "执行失败: {{.msg}}",
	DataSyncingMsg:              "执行中，已同步{{.count}}条",
	DataSyncValidationMsg:       "数据校验：源={{.srcCount}}条，目标={{.targetCount}}条，差异={{.diff}}条",
	DataSyncBiDirReverseCreated: "双向同步：已为正向任务[{{.forwardTaskId}}]创建反向任务[{{.reverseTaskId}}]",

	// db mask
	LogDbMaskRuleSave:        "mask-保存脱敏规则",
	LogDbMaskRuleDelete:      "mask-删除脱敏规则",
	LogDbMaskTagSave:         "mask-保存脱敏列标签",
	LogDbMaskTagDelete:       "mask-删除脱敏列标签",
	ErrMaskTagNeedAlgoOrRule: "绑定规则的列标签需指定脱敏算法或关联规则",

	// 定时调度
	ErrTaskCronInvalid: "定时调度的 cron 表达式【{{.cron}}】非法，任务不会按计划执行：{{.reason}}",
}
