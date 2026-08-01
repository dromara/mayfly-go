package i18n

// ============================================================
// 中文语言包
// ============================================================

var zhCN = map[string]string{
	// 通用
	MsgYes:      "是",
	MsgNo:       "否",
	MsgIndex:    "序号",
	MsgName:     "名称",
	MsgType:     "类型",
	MsgAddress:  "地址",
	MsgMode:     "模式",
	MsgStatus:   "状态",
	MsgTotal:    "总数",
	MsgOnline:   "在线",
	MsgOffline:  "离线",
	MsgInputIdx: "输入序号选择 {{.resource}}，或按 q 退出",
	MsgInvalid:  "无效的序号，请输入 1-{{.max}}",
	MsgQuit:     "退出",
	MsgBack:     "返回",
	MsgExit:     "退出",
	MsgNoData:   "暂无数据",

	MsgSelectOp:    "选择操作 (1-{{.max}})",
	MsgInvalidOp:   "无效的选择，请输入 1-{{.max}}",
	MsgOpFailed:    "操作失败",
	MsgOpSuccess:   "操作成功",
	MsgLoading:     "加载中...",
	MsgFailed:      "失败",
	MsgConfigSaved: "配置保存成功！",
	MsgConfigPath:  "配置文件位置: {{.path}}",
	MsgReadyToUse:  "现在可以开始使用了:",

	MsgCmdExecFailed: "命令执行失败",
	MsgArgNotNumber:  "参数必须为数字",

	// 全局标志
	MsgFlagTimeout:  "请求超时时间（秒）",
	MsgFlagName:     "资源名称（可替代 --id）",
	MsgFlagOutput:   "输出格式 (table/json/csv/yaml)",
	MsgFlagProfile:  "服务器配置档案名称",
	MsgFlagAgent:    "Agent 模式（禁用交互，JSON 输出，结构化错误码）",
	MsgFlagDryRun:   "预演模式（显示将执行的操作，不实际执行）",
	MsgFlagPage:     "页码（从 1 开始）",
	MsgFlagPageSize: "每页数量",
	MsgFlagIds:      "多个 ID（逗号分隔）",

	// Agent 命令
	MsgAgentCommandsShort: "列出所有可用命令（机器可读）",
	MsgAgentCommandsLong:  "以 JSON 格式输出所有命令及其参数，供 Agent 程序化调用",

	// 资源解析错误
	MsgResolveNotFound:    "{{.type}} '{{.name}}' 未找到",
	MsgResolveMultiple:    "多个 {{.type}} 匹配 '{{.name}}'，请使用精确名称或 ID",
	MsgResolveUnsupported: "不支持的资源类型: {{.type}}",
	MsgResolveCannotParse: "无法解析匹配的 {{.type}} 的 ID",

	MsgCfgHomeDirFailed:   "获取用户目录失败",
	MsgCfgReadFailed:      "读取配置文件失败",
	MsgCfgParseFailed:     "解析配置文件失败",
	MsgCfgSerializeFailed: "序列化配置失败",
	MsgCfgWriteFailed:     "写入配置文件失败",
	MsgCfgInvalidURL:      "服务器地址格式无效，必须以 http:// 或 https:// 开头",

	// Root / Dashboard
	MsgRootShort: "Mayfly-Go CLI - 资源连接与管理工具",
	MsgRootLong: `Mayfly-Go CLI 是一个命令行工具，用于连接和管理各种资源：
- 数据库（MySQL, PostgreSQL, Oracle 等）
- 机器（SSH 连接）
- Redis
- MongoDB

Agent 使用示例:
  mayfly-cli --json login -s http://host -u admin -p password
  mayfly-cli --json db list
  mayfly-cli --json db exec --id 1 --db mydb --sql "SELECT 1"
  mayfly-cli --json ssh list
  mayfly-cli --json redis exec --id 1 --db 0 --cmd "GET key"

使用 --json 标志可获得结构化 JSON 输出，便于程序解析。`,

	MsgDashboardTitle:      "🚀  Mayfly-Go CLI 资源概览",
	MsgDashboardServer:     "服务器:",
	MsgDashboardStatus:     "状态:",
	MsgDashboardLoggedIn:   "已登录",
	MsgDashboardNotLogged:  "未登录 - 请先运行以下命令登录:",
	MsgDashboardAssets:     "资产统计:",
	MsgDashboardType:       "类型",
	MsgDashboardIcon:       "图标",
	MsgDashboardTotalCol:   "总数",
	MsgDashboardAvailCol:   "可用",
	MsgDashboardDesc:       "说明",
	MsgDashboardQuickStart: "💡 快速开始:",
	MsgDashboardDb:         "数据库:",
	MsgDashboardMachine:    "机  器:",
	MsgDashboardRedis:      "Redis:",
	MsgDashboardConfig:     "配  置:",
	MsgDashboardOnlineFmt:  "在线 {{.count}} 台",
	MsgDashboardDbDesc:     "支持 MySQL/PostgreSQL/SQLite 等",
	MsgDashboardRedisDesc:  "支持 Standalone/Cluster/Sentinel",

	MsgLoginPromptTitle:  "⚠️  未登录 - 请先配置连接",
	MsgLoginPromptServer: "服务器地址:",
	MsgLoginPromptQuick:  "快速配置:",
	MsgLoginPromptManual: "或手动指定:",

	MsgAssetStatsFailed: "获取资产统计失败: {{.err}}",

	MsgDashboardDbName:      "数据库",
	MsgDashboardMachineName: "机器",
	MsgHelpersAuthRequired:  "认证失败，请先登录",
	MsgTokenExpired:         "登录已过期，请执行 mayfly-cli login 重新登录",

	// Logout / Whoami
	MsgLogoutShort:   "登出并清除本地凭证",
	MsgLogoutLong:    "清除本地存储的 token 和 refreshToken",
	MsgLogoutSuccess: "已成功登出",
	MsgLogoutFailed:  "登出失败",
	MsgWhoamiShort:   "显示当前登录状态",
	MsgWhoamiLong:    "显示当前服务器连接和认证状态",
	MsgWhoamiServer:  "服务器",
	MsgWhoamiStatus:  "状态",
	MsgWhoamiUser:    "用户",
	MsgWhoamiOnline:  "已登录",
	MsgWhoamiOffline: "未登录",

	// 配置 / 登录
	MsgConfigShort: "配置 CLI 连接信息",
	MsgConfigLong: `配置 mayfly-go 服务端地址和登录信息。

交互模式（默认）:
  mayfly-cli config

非交互模式（适合 agent）:
  mayfly-cli config --server http://localhost:8888 --username admin --password admin123.
  mayfly-cli --json config -s http://localhost:8888 -u admin -p admin123.`,

	MsgConfigWizardTitle: "Mayfly-Go CLI 首次配置向导",
	MsgConfigServerHint:  "请输入 mayfly-go 服务端地址 (例如: http://localhost:8889) [{{.server}}]: ",
	MsgConfigUserHint:    "请输入用户名 (默认: admin): ",
	MsgConfigPassHint:    "请输入密码 (或设置 MAYFLY_PASSWORD 环境变量): ",

	MsgLoginShort: "登录 mayfly-go 服务端并保存 token",
	MsgLoginLong: `登录 mayfly-go 服务端，获取并保存访问令牌。

Agent 使用:
  mayfly-cli --json login -s http://host -u admin -p password
  mayfly-cli login --server http://host --username admin --password password`,

	MsgLoginLogging:    "正在登录...",
	MsgLoginStep1:      "[1/3] 获取 RSA 公钥... ",
	MsgLoginStep2:      "[2/3] 加密密码... ",
	MsgLoginStep3:      "[3/3] 登录获取 Token... ",
	MsgLoginSuccess:    "登录成功",
	MsgLoginFailed:     "登录失败",
	MsgLoginGetPubKey:  "获取 RSA 公钥失败: {{.err}}",
	MsgLoginEncrypt:    "加密密码失败: {{.err}}",
	MsgLoginSaveCfg:    "保存配置失败: {{.err}}",
	MsgLoginNoPassword: "未提供密码，请通过 --password 参数或 MAYFLY_PASSWORD 环境变量指定",

	MsgConfigFlagPath: "配置文件路径",
	MsgConfigFlagUser: "用户名（提供则跳过交互提示）",
	MsgConfigFlagPass: "密码（提供则跳过交互提示）",
	MsgLoginFlagUser:  "用户名",
	MsgLoginFlagPass:  "密码",

	// 数据库命令
	MsgDbShort:     "数据库操作",
	MsgDbLong:      "管理数据库连接和执行 SQL 查询",
	MsgDbListShort: "列出可用数据库",
	MsgDbListLong: `列出可用数据库实例。

Agent 使用:
  mayfly-cli --json db list
  mayfly-cli db list  # 交互模式`,
	MsgDbListFailed:   "获取数据库列表失败: {{.err}}",
	MsgDbNoResource:   "没有找到数据库资源",
	MsgDbTableHeaders: "序号,名称,类型,地址",
	MsgDbTableTip:     "输入序号选择数据库，或按 q 退出",
	MsgDbExecShort:    "执行 SQL 语句（使用实际 ID）",
	MsgDbExecLong: `使用数据库实际 ID 执行 SQL 语句，适合 agent 和脚本调用。

示例:
  mayfly-cli --json db exec --id 1 --db mydb --sql "SELECT * FROM users LIMIT 10"
  mayfly-cli db exec --id 1 --db mydb --sql "SHOW TABLES"`,
	MsgDbExecRequired: "必须提供 --id, --db, --sql 三个参数",
	MsgDbExecFailed:   "SQL 执行失败",
	MsgDbSqlShort:     "进入 SQL 执行模式",
	MsgDbSqlLong: `进入指定数据库的 SQL 执行模式。

交互模式:
  mayfly-cli db sql        # 交互式选择
  mayfly-cli db sql 1      # 选择实例后交互

非交互模式（agent）:
  mayfly-cli --json db sql --id 1 --db mydb --sql "SELECT 1"
  mayfly-cli --json db sql --id 1 --db mydb  # 进入指定库执行单条 SQL`,
	MsgDbSqlInvalidIdx: "无效的数据库实例序号: {{.arg}}，请输入 1-{{.max}}",
	MsgDbSqlGetDbInfo:  "🔍 获取数据库实例 {{.name}} 的库信息...",
	MsgDbSqlNoDb:       "该实例下没有可用的数据库",
	MsgDbSqlSelectDb:   "📌 选择数据库 (输入序号)",
	MsgDbSqlDbList:     "📊 数据库实例 {{.name}} 下的数据库列表:",
	MsgDbSqlConnecting: "🔍 验证数据库连接...",
	MsgDbSqlConnected:  "✅ 数据库连接成功: {{.db}} ({{.type}})",
	MsgDbSqlConnFailed: "数据库连接失败: {{.err}}",
	MsgDbTablesShort:   "查看数据库表列表",
	MsgDbTablesLong: `查看指定数据库的表列表。

Agent 使用:
  mayfly-cli --json db tables --id 1 --db mydb`,
	MsgDbTablesRequired: "必须提供 --id 和 --db 参数",
	MsgDbTablesFailed:   "获取表列表失败: {{.err}}",
	MsgDbDescShort:      "查看表结构",
	MsgDbDescLong: `查看指定表的结构信息。

Agent 使用:
  mayfly-cli --json db desc --id 1 --db mydb --table users`,
	MsgDbDescRequired: "必须提供 --id, --db, --table 参数",
	MsgDbDescFailed:   "查询表结构失败: {{.err}}",

	MsgDbColumnsShort: "查看表列信息（含备注）",
	MsgDbColumnsLong: `查看表的列元数据信息，包含列名、类型、是否主键、默认值、备注等。

示例:
  mayfly-cli db columns --id 1 --db mydb --table users
  mayfly-cli --json db columns --id 1 --db mydb --table users`,
	MsgDbColumnsRequired: "必须提供 --id, --db, --table 参数",
	MsgDbColumnsFailed:   "查询列信息失败: {{.err}}",

	MsgDbDdlShort: "查看建表语句",
	MsgDbDdlLong: `查看表的 CREATE TABLE 语句。

示例:
  mayfly-cli db ddl --id 1 --db mydb --table users
  mayfly-cli --json db ddl --id 1 --db mydb --table users`,
	MsgDbDdlRequired: "必须提供 --id, --db, --table 参数",
	MsgDbDdlFailed:   "查询建表语句失败: {{.err}}",

	MsgDbIndexShort: "查看表索引信息",
	MsgDbIndexLong: `查看表的索引信息。

示例:
  mayfly-cli db index --id 1 --db mydb --table users
  mayfly-cli --json db index --id 1 --db mydb --table users`,
	MsgDbIndexRequired: "必须提供 --id, --db, --table 参数",
	MsgDbIndexFailed:   "查询索引信息失败: {{.err}}",

	MsgDbDumpShort: "导出数据库 SQL",
	MsgDbDumpLong: `导出数据库结构或数据为 SQL 文件。

type: 1=结构, 2=数据, 3=结构+数据

示例:
  mayfly-cli db dump --id 1 --db mydb --type 3
  mayfly-cli db dump --id 1 --db mydb --type 1 --tables users,orders -o ./backup.sql`,
	MsgDbDumpRequired: "必须提供 --id, --db, --type 参数",
	MsgDbDumpFailed:   "导出失败: {{.err}}",
	MsgDbDumpOk:       "✅ 已导出到",

	MsgDbMenuTitle:         "📊  数据库列表",
	MsgDbMenuGetInfo:       "获取数据库信息",
	MsgDbMenuExec:          "执行 SQL",
	MsgDbMenuTables:        "查看表列表",
	MsgDbMenuDesc:          "查看表结构",
	MsgDbMenuBack:          "返回数据库列表",
	MsgDbMenuExit:          "退出",
	MsgDbMenuSelect:        "📌 选择数据库实例 (输入序号，q 退出): ",
	MsgDbMenuInvalid:       "❌ 获取库信息失败: {{.err}}",
	MsgDbMenuGetInfoFailed: "获取库信息失败",
	MsgDbMenuNoDb:          "⚠️  该实例下没有可用的数据库",
	MsgDbMenuSelectDb:      "📌 选择数据库 (输入序号，b 返回，q 退出): ",
	MsgDbMenuConnTest:      "🔍 验证数据库连接...",
	MsgDbMenuConnFail:      "❌ 数据库连接失败: {{.err}}",
	MsgDbMenuConnOk:        "✅ 数据库连接成功: {{.db}} ({{.type}})",
	MsgDbMenuRetry:         "请选择其他数据库或按 q 退出",

	MsgDbSqlPrompt:        "💻 输入 SQL (输入 'back' 返回, 'quit' 退出):",
	MsgDbSqlHint1:         "提示: 支持 SQL 语句，如 SELECT * FROM table_name LIMIT 10",
	MsgDbSqlHint2:         "      输入 'history' 查看历史记录，'clear' 清空历史记录",
	MsgDbSqlHint3:         "      输入 'help' 查看常用 SQL 示例，'tables' 查看表列表",
	MsgDbSqlHint4:         "      输入 'desc <表名>' 查看表结构",
	MsgDbSqlGetTables:     "🔍 获取表列表...",
	MsgDbSqlGetInfo:       "🔍 获取数据库信息...",
	MsgDbSqlInfoTitle:     "📊 数据库信息:",
	MsgDbSqlInfoFailed:    "❌ 获取数据库信息失败: {{.err}}",
	MsgDbSqlDescSpecify:   "❌ 请指定表名，如: desc table_name",
	MsgDbSqlDescUnsupport: "⚠️  暂不支持 {{.type}} 数据库的表结构查询",
	MsgDbSqlQueryFailed:   "❌ 查询失败: {{.err}}",

	MsgDbHistoryNone:    "📝 暂无历史记录",
	MsgDbHistoryTitle:   "📝 SQL 历史记录 (最近 {{.count}} 条):",
	MsgDbHistoryCleared: "🗑️  历史记录已清空",

	MsgDbHelpTitle:       "📚 常用 SQL 示例:",
	MsgDbHelpQuery:       "📌 查询操作:",
	MsgDbHelpStruct:      "📌 表结构操作:",
	MsgDbHelpShortcut:    "📌 快捷命令:",
	MsgDbHelpAgent:       "💡 Agent 直接调用:",
	MsgDbHelpAgentExec:   "mayfly-cli --json db exec --id <ID> --db <db_name> --sql \"<SQL>\"",
	MsgDbHelpAgentTables: "mayfly-cli --json db tables --id <ID> --db <db_name>",
	MsgDbHelpAgentDesc:   "mayfly-cli --json db desc --id <ID> --db <db_name> --table <table_name>",

	MsgDbErrTitle:        "❌ SQL 执行失败",
	MsgDbErrConnRefused:  "🔌 连接被拒绝: 无法连接到数据库服务器",
	MsgDbErrNoDb:         "🗄️  数据库不存在: 指定的数据库名称无效",
	MsgDbErrAccessDenied: "🔒 访问被拒绝: 用户名或密码错误",
	MsgDbErrSyntax:       "📝 SQL语法错误: SQL语句格式不正确",
	MsgDbErrNoTable:      "📋 表不存在: 指定的表名无效",
	MsgDbErrTimeout:      "⏰ 执行超时: SQL查询执行时间过长",
	MsgDbErrDetail:       "❌ 错误详情: {{.err}}",

	MsgDbQueryOk:         "✅ 查询成功，无返回结果",
	MsgDbQueryRecords:    "✅ 共 {{.count}} 条记录",
	MsgDbQueryShowFirst:  "... (共 {{.count}} 条记录，显示前 100 条)",
	MsgDbGetDbListFailed: "获取数据库列表失败: {{.err}}",
	MsgDbConnFailed:      "连接失败: {{.err}}",
	MsgDbConnAnomaly:     "连接异常: 无返回结果",
	MsgDbSelectIdx:       "📌 选择数据库 (输入序号，b 返回，q 退出): ",

	MsgDbSqlUse:                    "sql [序号]",
	MsgDbSqlGetListFail:            "获取数据库列表失败",
	MsgDbSqlNoRes:                  "没有找到数据库资源",
	MsgDbSqlGetInfoFail:            "获取库信息失败",
	MsgDbSqlInvalidDbIdx:           "无效的数据库实例序号: {{.arg}}，请输入 1-{{.max}}",
	MsgDbSqlConnFail:               "数据库连接失败",
	MsgDbNonInteractiveGetListFail: "获取数据库列表失败",
	MsgDbNonInteractiveConnFail:    "数据库连接失败",
	MsgDbNonInteractiveExecFail:    "执行 SQL 失败",
	MsgDbNonInteractiveConnOk:      "✅ 数据库连接成功: {{.db}} (id={{.id}})",
	MsgDbSqlResultOk:               "✅ 查询成功，无返回结果",
	MsgDbFlagExecId:                "数据库实例 ID（从 db list 获取）",
	MsgDbFlagExecDb:                "数据库名称",
	MsgDbFlagExecSql:               "要执行的 SQL 语句",
	MsgDbFlagSqlId:                 "数据库实例 ID",
	MsgDbFlagSqlDb:                 "数据库名称",
	MsgDbFlagSqlSql:                "要执行的 SQL 语句（可选，不提供则仅测试连接）",
	MsgDbFlagTablesId:              "数据库实例 ID",
	MsgDbFlagTablesDb:              "数据库名称",
	MsgDbFlagDescId:                "数据库实例 ID",
	MsgDbFlagDescDb:                "数据库名称",
	MsgDbFlagDescTable:             "表名",
	MsgDbFlagDescType:              "数据库类型 (mysql/postgresql/sqlite/dm)",
	MsgDbFlagColumnsId:             "数据库实例 ID（从 db list 获取）",
	MsgDbFlagColumnsDb:             "数据库名称",
	MsgDbFlagColumnsTable:          "表名",
	MsgDbFlagDdlId:                 "数据库实例 ID（从 db list 获取）",
	MsgDbFlagDdlDb:                 "数据库名称",
	MsgDbFlagDdlTable:              "表名",
	MsgDbFlagIndexId:               "数据库实例 ID（从 db list 获取）",
	MsgDbFlagIndexDb:               "数据库名称",
	MsgDbFlagIndexTable:            "表名",
	MsgDbFlagDumpId:                "数据库实例 ID（从 db list 获取）",
	MsgDbFlagDumpDb:                "数据库名称",
	MsgDbFlagDumpType:              "导出类型: 1=结构, 2=数据, 3=结构+数据",
	MsgDbFlagDumpTables:            "指定表名（逗号分隔，可选）",
	MsgDbFlagDumpOutput:            "输出文件路径（默认输出到 stdout）",
	MsgDbHelpTables:                "tables                    -- 查看所有表",
	MsgDbHelpDesc:                  "desc table_name           -- 查看表结构",
	MsgDbHelpHelp:                  "help     -- 显示此帮助信息",
	MsgDbHelpHistory:               "history  -- 查看SQL历史记录",
	MsgDbHelpTablesList:            "tables   -- 查看表列表",
	MsgDbHelpDescTable:             "desc <表名> -- 查看表结构",
	MsgDbHelpBack:                  "back     -- 返回数据库列表",
	MsgDbHelpQuit:                  "quit     -- 退出程序",

	// SSH 命令
	MsgSshShort:     "SSH 连接管理",
	MsgSshLong:      "管理机器 SSH 连接和执行远程命令",
	MsgSshListShort: "列出可用机器",
	MsgSshListLong: `列出可用机器实例。

Agent 使用:
  mayfly-cli --json ssh list
  mayfly-cli ssh list  # 交互模式`,
	MsgSshListFailed:   "获取机器列表失败: {{.err}}",
	MsgSshNoResource:   "没有找到机器资源",
	MsgSshTableHeaders: "序号,名称,地址,状态",
	MsgSshTableTip:     "输入序号选择机器，或按 q 退出",
	MsgSshStatsShort:   "查看机器统计信息",
	MsgSshStatsLong: `查看指定机器的 CPU/内存/磁盘统计信息。

Agent 使用:
  mayfly-cli --json ssh stats --id 1`,
	MsgSshStatsRequired: "必须提供 --id 参数",
	MsgSshStatsFailed:   "获取统计信息失败: {{.err}}",
	MsgSshStatsTitle:    "📊 机器统计信息 (ID: {{.id}})",
	MsgSshProcessShort:  "查看机器进程列表",
	MsgSshProcessLong: `查看指定机器的进程列表。

Agent 使用:
  mayfly-cli --json ssh process --id 1`,
	MsgSshProcessRequired: "必须提供 --id 参数",
	MsgSshProcessFailed:   "获取进程列表失败: {{.err}}",
	MsgSshProcessTitle:    "📋 进程列表 (ID: {{.id}})",
	MsgSshUsersShort:      "查看机器用户列表",
	MsgSshUsersLong: `查看指定机器的用户列表。

Agent 使用:
  mayfly-cli --json ssh users --id 1`,
	MsgSshUsersRequired: "必须提供 --id 参数",
	MsgSshUsersFailed:   "获取用户列表失败: {{.err}}",
	MsgSshUsersTitle:    "👥 用户列表 (ID: {{.id}})",
	MsgSshUsersNoData:   "暂无用户数据",
	MsgSshUsersCount:    "共 {{.count}} 个用户",
	MsgSshExecShort:     "在远程机器上执行命令",
	MsgSshExecLong: `在远程机器上非交互式执行命令并返回结果。

Agent 使用:
  mayfly-cli --json ssh exec --id 1 --cmd "ls -la"
  mayfly-cli --json ssh exec --id 1 --cmd "df -h" --auth-cert certName`,
	MsgSshExecRequired:    "必须提供 --id 参数",
	MsgSshExecCmdRequired: "必须提供 --cmd 参数",
	MsgSshExecNotFound:    "未找到 ID={{.id}} 的机器或无法获取认证凭证",
	MsgSshExecFailed:      "命令执行失败: {{.err}}",
	MsgSshExecTitle:       "📋 命令执行结果 (机器ID: {{.id}})",
	MsgSshExecErrLabel:    "⚠️  执行错误: {{.err}}",
	MsgSshConnectShort:    "连接 SSH 终端",
	MsgSshConnectLong: `通过 WebSocket 连接到远程机器的 SSH 终端。

Agent 使用:
  mayfly-cli ssh connect --id 1
  mayfly-cli ssh connect --id 1 --auth-cert certName`,
	MsgSshConnectRequired: "必须提供 --id 参数",
	MsgSshConnectNotFound: "未找到 ID={{.id}} 的机器或无法获取认证凭证",
	MsgSshConnecting:      "🚀 正在连接 SSH 终端 (机器ID: {{.id}})...",
	MsgSshConnectFailed:   "SSH 终端连接失败: {{.err}}",
	MsgSshConnectClosed:   "✅ SSH 会话已关闭",

	MsgSshMenuTitle:    "🛠️   {{.name}} 操作菜单",
	MsgSshMenuTerminal: "SSH 终端",
	MsgSshMenuSysInfo:  "查看系统信息",
	MsgSshMenuProcess:  "查看进程列表",
	MsgSshMenuUsers:    "查看用户列表",
	MsgSshMenuBack:     "返回机器列表",
	MsgSshMenuExit:     "退出",
	MsgSshMenuSelect:   "📌 选择机器 (输入序号，q 退出): ",
	MsgSshMenuInvalid:  "❌ 无效的序号，请输入 1-{{.max}}",
	MsgSshSelected:     "✅ 已选择: {{.name}}",
	MsgSshOffline:      "⚠️  机器 [{{.name}}] 离线，无法操作",

	MsgSshProcessHeader: "PID,用户,CPU%,MEM%,命令",
	MsgSshProcessNoData: "暂无进程数据",
	MsgSshProcessMore:   "... (共 {{.count}} 行，仅显示前 25 行)",
	MsgSshProcessTotal:  "... (共 {{.count}} 个进程)",

	MsgSshUsersHeader: "用户名,组,Shell",
	MsgSshNoCert:      "⚠️  无法获取认证凭证信息",

	MsgSshFlagStatsId:     "机器 ID（从 ssh list 获取）",
	MsgSshFlagProcessId:   "机器 ID",
	MsgSshFlagUsersId:     "机器 ID",
	MsgSshFlagExecId:      "机器 ID",
	MsgSshFlagExecCmd:     "要执行的命令",
	MsgSshFlagExecCert:    "认证凭证名称（可选，不指定则自动获取）",
	MsgSshFlagConnectId:   "机器 ID",
	MsgSshFlagConnectCert: "认证凭证名称（可选，不指定则自动获取）",

	MsgSshSysInfoTitle:   "📊  {{.name}} 系统信息",
	MsgSshSysInfoGetFail: "❌ 获取失败: {{.err}}",
	MsgSshSysInfoCpu:     "CPU 使用率:",
	MsgSshSysInfoMem:     "内存使用率:",
	MsgSshSysInfoDisk:    "磁盘使用率:",
	MsgSshSysInfoLoad:    "系统负载 (1分钟):",

	MsgSshProcessGetFail:   "❌ 获取失败: {{.err}}",
	MsgSshProcessNoData2:   "暂无进程数据",
	MsgSshUserListTitle:    "👥  {{.name}} 用户列表",
	MsgSshUserListGetFail:  "❌ 获取失败: {{.err}}",
	MsgSshUserListNoData:   "暂无用户数据",
	MsgSshUserListHeader:   "用户名,组,Shell",
	MsgSshUserListCount:    "共 {{.count}} 个用户",
	MsgSshTerminalConnFail: "❌ 终端连接失败: {{.err}}",
	MsgSshTerminalClosed:   "✅ 终端会话已关闭",

	// SSH 文件操作
	MsgSshLsShort: "列出远程目录内容",
	MsgSshLsLong: `列出远程机器指定目录的文件和子目录。

示例:
  mayfly-cli ssh ls --id 1 --path /etc
  mayfly-cli --json ssh ls --id 1 --path /var/log`,
	MsgSshLsRequired: "必须提供 --id, --path 参数",
	MsgSshLsFailed:   "列出目录失败: {{.err}}",
	MsgSshCatShort:   "读取远程文件内容",
	MsgSshCatLong: `读取远程机器上的文件内容（文件需小于 1MB）。

示例:
  mayfly-cli ssh cat --id 1 --path /etc/hosts
  mayfly-cli --json ssh cat --id 1 --path /etc/nginx/nginx.conf`,
	MsgSshCatRequired:   "必须提供 --id, --path 参数",
	MsgSshCatFailed:     "读取文件失败: {{.err}}",
	MsgSshDownloadShort: "下载远程文件到本地",
	MsgSshDownloadLong: `从远程机器下载文件到本地。

示例:
  mayfly-cli ssh download --id 1 --path /var/log/app.log
  mayfly-cli ssh download --id 1 --path /etc/nginx/nginx.conf -o ./conf/`,
	MsgSshDownloadRequired:    "必须提供 --id, --path 参数",
	MsgSshDownloadFailed:      "下载失败: {{.err}}",
	MsgSshDownloadWriteFailed: "写入本地文件失败: {{.err}}",
	MsgSshDownloadOk:          "✅ 已下载",
	MsgSshUploadShort:         "上传本地文件到远程机器",
	MsgSshUploadLong: `将本地文件上传到远程机器指定目录。

示例:
  mayfly-cli ssh upload --id 1 --path /tmp/ --file ./deploy.sh
  mayfly-cli --json ssh upload --id 1 --path /opt/app/ --file ./app.jar`,
	MsgSshUploadRequired:   "必须提供 --id, --path, --file 参数",
	MsgSshUploadReadFailed: "读取本地文件失败: {{.err}}",
	MsgSshUploadFailed:     "上传失败: {{.err}}",
	MsgSshUploadOk:         "✅ 已上传",
	MsgSshWriteShort:       "写入远程文件内容",
	MsgSshWriteLong: `将内容写入远程机器的指定文件。

示例:
  mayfly-cli ssh write --id 1 --path /tmp/test.txt --content "hello world"`,
	MsgSshWriteRequired: "必须提供 --id, --path, --content 参数",
	MsgSshWriteFailed:   "写入失败: {{.err}}",
	MsgSshWriteOk:       "✅ 已写入",
	MsgSshRmShort:       "删除远程文件/目录",
	MsgSshRmLong: `删除远程机器上的文件或目录。

示例:
  mayfly-cli ssh rm --id 1 --path /tmp/test.txt`,
	MsgSshRmRequired: "必须提供 --id, --path 参数",
	MsgSshRmFailed:   "删除失败: {{.err}}",
	MsgSshRmOk:       "✅ 已删除",
	MsgSshKillShort:  "终止远程进程",
	MsgSshKillLong: `终止远程机器上的指定进程（kill -9）。

示例:
  mayfly-cli ssh kill --id 1 --pid 12345`,
	MsgSshKillRequired: "必须提供 --id, --pid 参数",
	MsgSshKillFailed:   "终止进程失败: {{.err}}",
	MsgSshKillOk:       "✅ 已终止",

	MsgSshFlagId:        "机器 ID（从 ssh list 获取）",
	MsgSshFlagPath:      "远程文件/目录路径",
	MsgSshFlagAuthCert:  "认证凭证名称（可选，默认自动获取）",
	MsgSshFlagOutput:    "本地保存路径（默认为当前目录）",
	MsgSshFlagRemoteDir: "远程目标目录",
	MsgSshFlagLocalFile: "本地文件路径",
	MsgSshFlagContent:   "要写入的内容",
	MsgSshFlagPid:       "要终止的进程 PID",
	MsgSshFlagSrc:       "源路径",
	MsgSshFlagDst:       "目标路径",
	MsgSshFlagName:      "新文件名",

	// SSH 文件操作
	MsgSshCpShort:     "复制远程文件或目录",
	MsgSshCpLong:      "在远程机器上复制文件或目录",
	MsgSshCpOk:        "已复制",
	MsgSshMvShort:     "移动/重命名远程文件或目录",
	MsgSshMvLong:      "在远程机器上移动或重命名文件或目录",
	MsgSshMvOk:        "已移动",
	MsgSshRenameShort: "重命名远程文件（保持同目录）",
	MsgSshRenameOk:    "已重命名",
	MsgSshMkdirShort:  "创建远程目录",
	MsgSshMkdirLong:   "在远程机器上创建目录",
	MsgSshMkdirOk:     "已创建目录",
	MsgSshTouchShort:  "创建远程空文件",
	MsgSshTouchOk:     "已创建文件",
	MsgSshStatShort:   "查看文件/目录状态信息",
	MsgSshOpFailed:    "操作失败",
	MsgSshOpRequired:  "缺少必需参数",

	// Redis 命令
	MsgRedisShort:     "Redis 操作",
	MsgRedisLong:      "管理 Redis 连接和执行命令",
	MsgRedisListShort: "列出可用 Redis 实例",
	MsgRedisListLong: `列出可用 Redis 实例。

Agent 使用:
  mayfly-cli --json redis list`,
	MsgRedisListFailed:   "获取 Redis 列表失败: {{.err}}",
	MsgRedisNoResource:   "没有找到 Redis 资源",
	MsgRedisTableHeaders: "序号,名称,地址,模式",
	MsgRedisTableTip:     "输入序号选择 Redis，或按 q 退出",
	MsgRedisExecShort:    "执行 Redis 命令（使用实际 ID）",
	MsgRedisExecLong: `使用 Redis 实际 ID 执行命令，适合 agent 和脚本调用。

示例:
  mayfly-cli --json redis exec --id 1 --db 0 --cmd "GET mykey"
  mayfly-cli --json redis exec --id 1 --db 0 --cmd "SET foo bar"
  mayfly-cli --json redis exec --id 1 --db 0 --cmd "KEYS *"`,
	MsgRedisExecRequired: "必须提供 --id 和 --cmd 参数",
	MsgRedisExecFailed:   "执行 Redis 命令失败: {{.err}}",
	MsgRedisInfoShort:    "查看 Redis 服务器信息",
	MsgRedisInfoLong: `查看 Redis 服务器详细信息。

Agent 使用:
  mayfly-cli --json redis info --id 1`,
	MsgRedisInfoRequired: "必须提供 --id 参数",
	MsgRedisInfoFailed:   "查询失败: {{.err}}",
	MsgRedisInfoTitle:    "📊 Redis 服务器信息 (ID: {{.id}})",
	MsgRedisMemShort:     "查看 Redis 内存使用",
	MsgRedisMemLong: `查看 Redis 内存使用信息。

Agent 使用:
  mayfly-cli --json redis mem --id 1`,
	MsgRedisMemRequired: "必须提供 --id 参数",
	MsgRedisMemFailed:   "查询失败: {{.err}}",
	MsgRedisMemTitle:    "💾 Redis 内存使用 (ID: {{.id}})",

	MsgRedisMenuTitle:    "🔴  {{.name}} Redis 操作菜单",
	MsgRedisMenuExec:     "执行 Redis 命令",
	MsgRedisMenuInfo:     "查看服务器信息",
	MsgRedisMenuMem:      "查看内存使用",
	MsgRedisMenuKeyspace: "查看键空间统计",
	MsgRedisMenuBack:     "返回 Redis 列表",
	MsgRedisMenuExit:     "退出",
	MsgRedisMenuSelect:   "📌 选择 Redis (输入序号，q 退出): ",
	MsgRedisMenuInvalid:  "❌ 无效的序号，请输入 1-{{.max}}",
	MsgRedisSelected:     "✅ 已选择: {{.name}} ({{.mode}})",

	MsgRedisExecPrompt: "💻 输入 Redis 命令 (输入 'back' 返回, 'quit' 退出):",
	MsgRedisExecHint:   "提示: 格式为 'DB编号 命令'，例如 '0 KEYS *'",
	MsgRedisExecFmtErr: "❌ 格式错误，请使用: DB编号 命令",
	MsgRedisExecDbErr:  "❌ DB 编号无效: {{.db}}",
	MsgRedisExecFail:   "❌ 执行失败: {{.err}}",

	MsgRedisInfoVersion:   "版本:",
	MsgRedisInfoOs:        "系统:",
	MsgRedisInfoPort:      "端口:",
	MsgRedisInfoPid:       "进程ID:",
	MsgRedisInfoUptime:    "运行时间:",
	MsgRedisInfoQueryFail: "❌ 查询失败: {{.err}}",

	MsgRedisMemUsed:      "已使用:",
	MsgRedisMemMax:       "最大值:",
	MsgRedisMemPeak:      "峰值:",
	MsgRedisMemRss:       "RSS:",
	MsgRedisMemQueryFail: "❌ 查询失败: {{.err}}",

	MsgRedisKeyspaceTitle:     "🔑  {{.name}} 键空间统计",
	MsgRedisKeyspaceNoData:    "暂无键空间数据",
	MsgRedisKeyspaceHeader:    "数据库,键数量,过期时间",
	MsgRedisKeyspaceQueryFail: "❌ 查询失败: {{.err}}",

	MsgRedisFlagExecId:  "Redis 实例 ID（从 redis list 获取）",
	MsgRedisFlagExecDb:  "Redis DB 编号",
	MsgRedisFlagExecCmd: "要执行的 Redis 命令",
	MsgRedisFlagInfoId:  "Redis 实例 ID",
	MsgRedisFlagMemId:   "Redis 实例 ID",

	MsgRedisScanShort: "扫描 Redis Key 列表",
	MsgRedisScanLong: `使用 SCAN 命令扫描 Redis Key 列表，支持模式匹配。

示例:
  mayfly-cli redis scan --id 1 --db 0 --match "user:*" --count 100
  mayfly-cli --json redis scan --id 1 --db 0`,
	MsgRedisScanRequired:  "必须提供 --id 参数",
	MsgRedisScanFailed:    "扫描 Key 失败: {{.err}}",
	MsgRedisFlagScanId:    "Redis 实例 ID（从 redis list 获取）",
	MsgRedisFlagScanDb:    "Redis DB 编号",
	MsgRedisFlagScanMatch: "Key 匹配模式（默认 *）",
	MsgRedisFlagScanCount: "每次扫描数量（默认 100）",

	// Redis Key 操作
	MsgRedisKeyInfoShort: "查看键类型、值和 TTL",
	MsgRedisKeyInfoLong:  "检查 Redis 键的类型、值和 TTL",
	MsgRedisKeyTtlShort:  "查看键 TTL（秒）",
	MsgRedisKeyMemShort:  "查看键内存占用（字节）",
	MsgRedisKeyTtlResult: "TTL: {{.ttl}}",
	MsgRedisKeyMemResult: "内存: {{.mem}} 字节",
	MsgRedisFlagKeyId:    "Redis 实例 ID",
	MsgRedisFlagKeyDb:    "Redis 数据库编号",
	MsgRedisFlagKeyName:  "Redis 键名",

	// 终端通用
	MsgTermTip:          "💡 提示:",
	MsgTermNoResource:   "没有可用的 {{.type}} 资源",
	MsgTermSelectRes:    "请选择{{.type}}序号 (1-{{.max}}): ",
	MsgTermInputInvalid: "输入无效: {{.err}}",
	MsgTermOutOfRange:   "序号超出范围 (1-{{.max}})",

	// 客户端内部错误
	MsgClientSerializeFailed:    "序列化请求体失败",
	MsgClientCreateReqFailed:    "创建请求失败",
	MsgClientSendReqFailed:      "发送请求失败",
	MsgClientReadRespFailed:     "读取响应失败",
	MsgClientRequestFailed:      "请求失败",
	MsgClientParseRespFailed:    "解析响应数据失败",
	MsgClientEncryptSqlFailed:   "加密 SQL 失败",
	MsgClientConnFailed:         "连接失败",
	MsgClientConnAnomaly:        "连接异常: 无返回结果",
	MsgClientGetInstDbsFailed:   "获取实例数据库列表失败",
	MsgClientGetDbInstFailed:    "获取数据库实例信息失败",
	MsgClientNoCertInfo:         "未找到认证凭证信息",
	MsgClientGetTableFailed:     "获取表信息失败",
	MsgClientGetDbInfoFailed:    "获取数据库信息失败",
	MsgClientAesKeyTooShort:     "AES key 长度不足: 需要 24 字节, 实际 {{.len}} 字节",
	MsgClientAesCipherFailed:    "创建 AES 实例失败",
	MsgClientTerminalConnFailed: "连接终端失败",
	MsgClientTerminalModeFailed: "设置终端模式失败",
	MsgClientHttpFailed:         "HTTP 请求失败",
	MsgClientHttpStatus:         "HTTP 状态码: {{.code}}",
	MsgClientReadRespFailed2:    "读取响应失败",
	MsgClientParseRespFailed2:   "解析响应失败",
	MsgClientServerError:        "服务器错误: {{.msg}}",
	MsgClientNoPublicKey:        "响应中未找到公钥",
	MsgClientParsePEMFailed:     "无法解析 PEM 格式的公钥",
	MsgClientParseKeyFailed:     "解析公钥失败",
	MsgClientNotRSAKey:          "不是 RSA 公钥",
	MsgClientEncryptFailed:      "加密失败",
	MsgClientSerializeFailed2:   "序列化请求失败",
	MsgClientHttpReqFailed:      "HTTP 请求失败",
	MsgClientReadRespFailed3:    "读取响应失败",
	MsgClientParseRespFailed3:   "解析响应失败",
	MsgClientLoginFailed:        "登录失败: {{.msg}}",
	MsgClientRespFormatErr:      "响应数据格式错误",
	MsgClientNoToken:            "未找到 token 字段",

	// MongoDB
	MsgMongoShort:          "MongoDB 操作",
	MsgMongoLong:           "管理 MongoDB 连接、浏览数据库/集合、执行查询",
	MsgMongoListShort:      "列出 MongoDB 实例",
	MsgMongoNoResource:     "未找到 MongoDB 实例",
	MsgMongoDbShort:        "列出数据库",
	MsgMongoColsShort:      "列出集合",
	MsgMongoFindShort:      "查询集合文档",
	MsgMongoFindLong:       "使用 JSON 过滤器查询集合中的文档",
	MsgMongoRunShort:       "执行 MongoDB 命令 (JSON)",
	MsgMongoRunLong:        "执行原始 MongoDB 命令",
	MsgMongoTableTip:       "使用: mayfly-cli mongo databases --id <ID>",
	MsgMongoFlagId:         "MongoDB 实例 ID",
	MsgMongoFlagDb:         "数据库名",
	MsgMongoFlagCollection: "集合名",
	MsgMongoFlagFilter:     "查询过滤器 (JSON)",
	MsgMongoFlagLimit:      "最大返回文档数",
	MsgMongoFlagSkip:       "跳过文档数",
	MsgMongoFlagCommand:    "MongoDB 命令 (JSON)",
	MsgMongoInvalidFilter:  "无效的 --filter JSON: {{.err}}",
	MsgMongoInvalidCommand: "无效的 --command JSON: {{.err}}",

	// Ping
	MsgPingShort:    "检查服务器连通性和认证状态",
	MsgPingLong:     "验证 mayfly-go 服务器是否可达以及 token 是否有效",
	MsgPingFailed:   "Ping 失败: {{.err}}",
	MsgPingTitle:    "Mayfly-Go 服务器状态",
	MsgPingServer:   "服务器",
	MsgPingAuth:     "认证",
	MsgPingLatency:  "延迟",
	MsgPingAuthOk:   "已认证",
	MsgPingAuthAnon: "匿名",

	// DB Version
	MsgDbVersionShort:  "查看数据库服务器版本",
	MsgDbVersionLong:   "获取数据库实例的版本信息",
	MsgDbVersionResult: "数据库 ID {{.id}} ({{.db}}) 版本: {{.version}}",
	MsgDbFlagVersionId: "数据库实例 ID",
	MsgDbFlagVersionDb: "数据库名",
}
