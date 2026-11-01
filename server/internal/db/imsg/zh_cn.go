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

	LogDbTableAclSave:   "DB-保存表访问控制",
	LogDbTableAclDelete: "DB-删除表访问控制",

	LogDbColumnAclSave:   "DB-保存列访问控制",
	LogDbColumnAclDelete: "DB-删除列访问控制",

	ErrExportTicketNotFound:      "找不到该导出工单",
	ErrExportTicketMismatch:      "该工单不是导出工单，不能用于下载",
	ErrExportTicketNotApproved:   "该导出工单尚未审批通过",
	ErrExportTicketExpired:       "该导出工单已超出可下载时限（审批通过后 24 小时内有效）",
	ErrExportTicketNotOwner:      "只有工单申请人本人可以使用这张导出工单",
	ErrExportTicketInvalid:       "工单里没有记录有效的导出范围（库与表清单），不能作为下载凭据",
	ErrExportTicketScopeMismatch: "本次下载的库或表与工单批准的范围不一致",

	ErrAclUnresolvable:      "该库启用了表/列访问控制，而这条语句的表引用无法静态确定（派生表、子查询或解析失败），已拒绝执行",
	ErrAccountsUnreadable:   "读不到该实例的账号信息（多为账号缺少只读权限）：{{.reason}}",
	ErrAccountsUnsupported:  "该实例无法盘点账号：{{.reason}}",
	ErrRecoveryUnsupported:  "该实例无法探测恢复能力：{{.reason}}",
	ErrDdlPrecheckTables:    "预检不完整：读取表规模失败（{{.err}}）",
	ErrMetricUnsupported:    "不支持该指标：{{.metric}}",
	ErrPerfUnsupported:      "当前数据库类型未实现该项性能诊断能力",
	ErrSlowQueryUnavailable: "该实例无法提供慢查询统计：{{.reason}}",
	ErrLockWaitUnavailable:  "该实例无法提供锁等待链：{{.reason}}。请确认 performance_schema 已开启且版本支持",

	ErrColumnAccessDenied:      "无权访问列【{{.column}}】（已被管理员列入保护名单）",
	ErrWholeTableReadDenied:    "该表存在无权查看的列（{{.columns}}），整表导出/复制会绕过列保护；请改用查询导出并显式选择列",
	ErrColumnAclColumnRequired: "请选择要保护的列",

	ErrTableAccessDenied:        "无权访问表【{{.table}}】（已被管理员列入保护名单）",
	ErrTableAclRolesUnavailable: "无法确认当前账号的角色，表级权限按拒绝处理",
	ErrTableAclTableRequired:    "请选择要保护的表",
	ErrTableNotAllowed:          "无权访问表【{{.table}}】（该库启用了访问白名单，该表不在你的可访问清单中）",

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

	// 迁移导出文件
	LogDtsDownloadFile: "dts-下载迁移导出文件",

	// 原子批次执行
	ErrAtomicUnsupportedStmt: "原子执行只支持 INSERT/UPDATE/DELETE，语句【{{.sql}}】不属于这三类：DDL 在部分数据库会隐式提交、无法回滚，查询结果也不参与回滚",
	ErrAtomicBatchRolledBack: "第 {{.index}}/{{.total}} 条语句执行失败，整批已回滚（前面执行成功的语句同样未生效）：{{.error}}",

	ErrSqlExecNotFoundToCancel: "未找到进行中的该执行，无法取消（可能早已结束、落在其他服务实例，或服务进程已重启）",
	ErrSqlExecJustFinished:     "该执行刚刚结束，无需取消",
	ErrSqlExecIdConflict:       "执行标识与他人进行中的执行冲突，本次执行已拒绝，请重新发起",
	ErrSqlExecNotOwned:         "只能取消自己发起的执行",

	TriggerReasonRowsExceed:    "估算影响行数达到或超过阈值（或该数据库无法给出估算）",
	TriggerReasonFullScan:      "执行计划显示存在全表/全索引扫描",
	TriggerReasonBigTableDdl:   "目标表数据量达到大表阈值，结构变更需额外确认锁表与耗时风险",
	TriggerReasonNoPrimaryKey:  "目标表没有主键",
	TriggerReasonSelectStar:    "查询使用 SELECT * 拉取全部列",
	TriggerReasonMultiTableDml: "多表更新/删除无法按行定位旧值",
	TriggerReasonTableInList:   "目标表命中管理员设定的保护名单",
	ErrReviewCheckNotEvaluated: "该检查项本次无法给出结论，不代表未命中：{{.reason}}",
	ErrReviewNoSqlToAnalyze:    "缺少可分析的语句",
	ErrReviewNoTargetDb:        "预检必须指定目标数据库，否则无法按该库方言解析与估算",
	ErrReviewNoTargetTable:     "无法从语句中确定目标表，取不到表规模",
	ErrReviewNoTableStats:      "目标表的统计信息不可获取，无法判断表规模",
	ReviewEstimateUnavailable:  "本次拿不到行数估算：{{.reason}}（不等于影响 0 行，也不表示该语句安全）",

	LogDbCancelSession: "db-取消数据库会话",
	LogDbCancelExec:    "db-取消SQL执行",

	ErrSessionCancelFailed: "取消会话 {{.sessionId}} 失败（该会话可能已结束或语句已跑完）：{{.reason}}",

	LogDbGenRollback: "db-生成回滚SQL",

	// 回滚 SQL 生成：每条失败都必须说清「为什么给不出」，而不是一句不支持
	ErrRollbackRecordNotFound:   "找不到该 SQL 执行记录（id={{.id}}），无法取到当时采集的旧值",
	ErrRollbackNotExecuted:      "该 SQL 执行记录（id={{.id}}）不是一次执行成功的变更，旧值对应的改动并未生效，生成回滚会把陈旧值写回现网",
	ErrRollbackUnsupportedType:  "只有 UPDATE、DELETE 与 INSERT 的执行记录携带了可用于回滚的行数据，该记录类型为【{{.type}}】",
	ErrRollbackNoOldValue:       "该执行记录没有旧值：要么当时采集中失败（见服务端日志），要么语句无法按行定位，因此无法生成回滚 SQL",
	ErrRollbackOldValueInvalid:  "记录里的行数据无法解析为行集合，未生成回滚 SQL：{{.reason}}",
	ErrRollbackNoTable:          "目标表【{{.table}}】已不存在或不可读，无法按旧值结构生成回滚 SQL",
	ErrRollbackColumnGone:       "采集到的列【{{.column}}】在表【{{.table}}】中已不存在，强行生成会让列值错位，未生成回滚 SQL",
	ErrRollbackPkChanged:        "该语句修改了主键列【{{.column}}】，旧值里只有主键的原值，无法定位改后的行，因此无法生成回滚 SQL",
	RollbackTruncatedNote:       "行数据采集上限为 {{.cap}} 行，当时超出部分未被记录，这里只生成可覆盖的行",
	ErrRollbackNoPrimaryKey:     "表【{{.table}}】没有主键，行数据无法唯一定位；按无主键条件生成回滚语句会波及同值的其他行，因此未生成回滚 SQL",
	RollbackStmtUnparsedNote:    "原语句无法解析，若该 UPDATE 修改过主键列，这里只还原非主键列的值（主键还原需人工处理）",
	RollbackUnverifiedNote:      "该记录没有执行后的行快照（升级前记录或当时采集失败），本次未做并发校验：回滚可能覆盖这些行在此之后的修改",
	RollbackBlockedNote:         "其中 {{.count}} 行在变更之后又被修改、删除或无法定位，未生成它们的回滚，需人工对账",
	ErrRollbackNoInsertSnapshot: "该 INSERT 记录没有采集到插入值（语句形态不支持或当时解析失败），无法定位新插入的行，因此无法生成回滚 SQL",

	ErrSchemaDiffSameDb:        "结构对比必须选两个不同的库：同一个库自比只会得到「无差异」，那不是对比结果",
	ErrSchemaDiffNoTables:      "两侧都没有可对比的表（表清单为空，或当前账号读不到任何表）",
	ErrSchemaDiffTooManyTables: "一次最多对比 {{.max}} 张表，请收窄表清单",
	SchemaDiffTruncatedNote:    "未指定表清单，只对比前 {{.max}} 张表（两侧共 {{.total}} 张）；其余表未参与本次对比",

	LogDbExportQuery: "db-导出查询结果集",

	ErrQueryExportEmptySql:          "没有可导出的查询语句",
	ErrQueryExportReadOnlyOnly:      "导出会重新执行这条语句，因此只允许查询类语句，当前语句类型为【{{.type}}】",
	ErrQueryExportFormatUnsupported: "导出格式【{{.format}}】不适用于查询结果集，可选格式：{{.supported}}",

	LogDbExplainPlan: "db-取SQL执行计划",

	// 拿不到计划不等于该语句没有代价：界面必须把这一句显示出来，而不是画一张空图或只列 0 个节点
	ErrExplainUnavailable:   "本次拿不到执行计划：{{.reason}}（不等于该语句无需计划，也不表示它安全）",
	ErrTLSStatusUnavailable: "无法读取本连接的加密状态：{{.reason}}（这不等于未加密，也不等于已加密）",

	// 表/列不存在时报不存在，而不是让 SQL 报错把用户输入原样回显
	ErrColumnValuesNoColumn: "表【{{.table}}】不存在列【{{.column}}】（或已被删除），无法采样该列的值",

	// 表结构漂移基线：截断与「读不到」都必须说出口，不能拿一份不完整的基线报「无漂移」
	ErrSchemaBaselineEmpty:      "该库当前读不到任何表结构，无法建立基线",
	SchemaBaselineTruncatedNote: "基线只覆盖前 {{.max}} 张表（该库共 {{.total}} 张）；其余表的结构变化不会被报为漂移",
	SchemaDriftColumnCount:      "列数 {{.from}}→{{.to}}",
	SchemaDriftPrimaryKeys:      "主键 {{.from}}→{{.to}}",
	SchemaDriftIndexCount:       "索引数 {{.from}}→{{.to}}",
	SchemaDriftColumnDefs:       "列定义或列顺序发生变化",
	SchemaDriftNoPk:             "无主键",
	SchemaDriftTableAdded:       "基线中不存在的新表",
	SchemaDriftTableRemoved:     "基线里有、当前库已不存在",
	LogDbSchemaBaselineSave:     "db-建立表结构基线",
	LogDbSchemaBaselineDel:      "db-删除表结构基线",
}
