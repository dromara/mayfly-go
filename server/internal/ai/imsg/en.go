package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	InfoIncomplete:          "Information incomplete, please complete",
	ParamCompletionTitle:    "Parameter Completion",
	ApprovalTitle:           "High-risk Operation Approval",
	ApprovalDesc:            "This operation requires approval before execution",
	RejectReasonDefault:     "User did not provide a specific reason",
	MissingRequiredParams:   "Missing required parameters",
	SqlExecApprovalReason:   "Executing SQL is a high-risk operation, please approve",
	ExecSqlToolDesc:         "ExecSql【Database SQL Execution】",
	ExecSqlToolInfo:         "[Database] SQL Execution - Execute non-query SQL statements (such as INSERT, UPDATE, DELETE, etc.). Applicable to data modification scenarios. Note: modification only, prohibit executing SELECT, SHOW, DESC, EXPLAIN and other query SQL. The remark parameter is required on every call: briefly state the purpose and expected effect of this SQL.",
	DbQueryDataToolDesc:     "DbQueryData【Database SQL Query】",
	DbQueryDataToolInfo:     "[Database] SQL Query - Batch execute read-only SQL statements (such as SELECT, SHOW, DESC, EXPLAIN, etc.). Applicable to querying table data, analyzing execution plans, etc. Supports querying multiple SQL statements at once (maximum 10) to avoid frequent calls. Note: query only, prohibit executing INSERT, UPDATE, DELETE and other modification SQL.\n\n[Important] If the user does not specify a database ID (dbId), please call this tool directly and provide an array of SQL statements. The system will automatically pop up the asset selection interface for the user to choose the database. Do not ask the user for the database ID!",
	DbQueryTableDDLToolDesc: "DbQueryTableDDL【Query Database Table DDL】",
	DbQueryTableDDLToolInfo: "[Database] Table DDL Query Tool - Batch query DDL definitions of specified tables, including complete metadata such as field names, data types, constraints, and indexes. Supports querying multiple tables at once. Applicable to understanding table structure before writing SQL, or checking table definitions when troubleshooting data issues.\n\n[Important] If the user does not specify a database ID (dbId), please call this tool directly and only provide the table name list. The system will automatically pop up the asset selection interface for the user to choose the database. Do not ask the user for the database ID!",
	DbInfoIncomplete:        "Missing database information, please complete the parameters",
	DbQueryTablesToolDesc:   "DbQueryTables【Query All Database Tables】",
	DbQueryTablesToolInfo:   "[Database] Table List Query Tool - Get basic information of all tables in the specified database, including table name, table comment, row count, data size, etc. Applicable to scenarios where you need to understand what tables exist in the database or quickly browse the table structure.\n\n[Important] If the user does not specify a database ID (dbId), please call this tool directly. The system will automatically pop up the asset selection interface for the user to choose the database. Do not ask the user for the database ID!",

	// Machine tools
	MachineCommandExecToolDesc: "MachineCommandExec【Remote Command Execution】",
	MachineCommandExecToolInfo: "[Machine] Remote Command Execution Tool - Execute commands on remote machines and return output results. Applicable to viewing system status, managing services, viewing logs, etc.\n\n[Command Usage Guide]\n1. View log files:\n   - Large log files (>1MB): Use `tail -n 100 /path/to/log` to view latest 100 lines, or `grep 'ERROR' /path/to/log | tail -n 50` to filter errors\n   - Search specific content: Use `grep -i 'error|exception|failed' /path/to/log` instead of cat\n   - Progressive analysis: Use `less /path/to/log` or `head -n 50 /path/to/log` to view in sections\n   - Count errors: Use `grep -c 'ERROR' /path/to/log`\n2. View large files: Use `head`, `tail`, `less` instead of `cat`\n3. Monitor real-time logs: Use `tail -f /path/to/log` (Note: will output continuously, need to stop manually)\n\n[Important] If the user does not specify an auth certificate name (authCertName), please call this tool directly and only provide the command. The system will automatically pop up the asset selection interface for the user to choose the machine. Do not ask the user for the auth certificate name!\n\n[Important] The remark parameter is required on every call: briefly state the purpose of this command so the user can understand and approve it.",
	MachineInfoIncomplete:      "Missing machine information, please complete the parameters",
	CommandExecApprovalReason:  "Executing commands may involve high-risk operations, please approve",
	InterruptExpired:           "The interrupt has expired or the service was restarted, please send a new message to start the task",
}
