package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	InfoIncomplete:          "信息不全，请完善",
	ParamCompletionTitle:    "参数补全",
	ApprovalTitle:           "高危操作审批",
	ApprovalDesc:            "该操作需要审批后才能执行",
	RejectReasonDefault:     "用户未提供具体原因",
	MissingRequiredParams:   "缺少必要参数",
	SqlExecApprovalReason:   "执行SQL是高危操作，请审批",
	ExecSqlToolDesc:         "ExecSql【数据库SQL执行】",
	ExecSqlToolInfo:         "【数据库】SQL执行 - 执行非查询类 SQL 语句（如 INSERT、UPDATE、DELETE 等）。适用于执行数据变更操作的场景。注意：仅限变更操作，禁止执行 SELECT、SHOW、DESC、EXPLAIN 等查询类 SQL。调用时必须在 remark 参数中简要说明本次执行的目的与预期效果。",
	DbQueryDataToolDesc:     "DbQueryData【数据库SQL查询】",
	DbQueryDataToolInfo:     "【数据库】SQL查询 - 批量执行只读类 SQL 语句（如 SELECT、SHOW、DESC、EXPLAIN 等）。适用于查询表数据、分析执行计划等场景。支持一次查询多条SQL（最多10条），避免频繁调用。注意：仅限查询操作，禁止执行 INSERT、UPDATE、DELETE 等变更类 SQL。\n\n【重要】如果用户没有指定数据库ID(dbId)，请直接调用此工具并提供SQL语句数组，系统会自动弹出资产选择界面让用户选择数据库。不要询问用户数据库ID！",
	DbQueryTableDDLToolDesc: "DbQueryTableDDL【查询数据库表DDL】",
	DbQueryTableDDLToolInfo: "【数据库】表DDL查询工具 - 批量获取指定数据表的 DDL 定义，包含字段名、数据类型、约束、索引等完整元数据。支持一次查询多个表。适用于编写 SQL 前了解表结构、排查数据问题时查看表定义等场景。\n\n【重要】如果用户没有指定数据库ID(dbId)，请直接调用此工具并只提供表名列表，系统会自动弹出资产选择界面让用户选择数据库。不要询问用户数据库ID！",
	DbInfoIncomplete:        "缺少数据库信息，请完善参数",
	DbQueryTablesToolDesc:   "DbQueryTables【查询数据库所有表】",
	DbQueryTablesToolInfo:   "【数据库】表列表查询工具 - 获取指定数据库中的所有表基本信息，包括表名、表备注、数据行数、数据大小等。适用于需要了解数据库有哪些表、快速浏览库内表结构的场景。\n\n【重要】如果用户没有指定数据库ID(dbId)，请直接调用此工具，系统会自动弹出资产选择界面让用户选择数据库。不要询问用户数据库ID！",

	// Machine tools
	MachineCommandExecToolDesc: "MachineCommandExec【远程命令执行】",
	MachineCommandExecToolInfo: "【机器】远程命令执行工具 - 在远程机器上执行命令并返回输出结果。适用于查看系统状态、管理服务、查看日志等场景。\n\n【命令使用指南】\n1. 查看日志文件：\n   - 大日志文件(>1MB)：使用 `tail -n 100 /path/to/log` 查看最新100行，或 `grep 'ERROR' /path/to/log | tail -n 50` 过滤错误\n   - 搜索特定内容：使用 `grep -i 'error|exception|failed' /path/to/log` 而不是 cat\n   - 渐进式分析：使用 `less /path/to/log` 或 `head -n 50 /path/to/log` 分段查看\n   - 统计错误数量：使用 `grep -c 'ERROR' /path/to/log`\n2. 查看大文件：使用 `head`、`tail`、`less` 而不是 `cat`\n3. 监控实时日志：使用 `tail -f /path/to/log` (注意：会持续输出，需要手动停止)\n\n【重要】如果用户没有指定授权凭证名称(authCertName)，请直接调用此工具并只提供命令，系统会自动弹出资产选择界面让用户选择机器。不要询问用户授权凭证名称！\n\n【重要】每次调用必须在 remark 参数中简要说明本次执行命令的目的，便于用户理解与审批。",
	MachineInfoIncomplete:      "缺少机器信息，请完善参数",
	CommandExecApprovalReason:  "执行命令可能涉及高危操作，请审批",
	InterruptExpired:           "该中断已失效或服务已重启，请重新发送消息发起任务",
}
