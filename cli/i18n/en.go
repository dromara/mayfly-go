package i18n

// ============================================================
// English language pack
// ============================================================

var en = map[string]string{
	// Common
	MsgYes:      "Yes",
	MsgNo:       "No",
	MsgIndex:    "No.",
	MsgName:     "Name",
	MsgType:     "Type",
	MsgAddress:  "Address",
	MsgMode:     "Mode",
	MsgStatus:   "Status",
	MsgTotal:    "Total",
	MsgOnline:   "Online",
	MsgOffline:  "Offline",
	MsgInputIdx: "Enter number to select {{.resource}}, or q to quit",
	MsgInvalid:  "Invalid number, please enter 1-{{.max}}",
	MsgQuit:     "Quit",
	MsgBack:     "Back",
	MsgExit:     "Exit",
	MsgNoData:   "No data",

	MsgSelectOp:    "Select operation (1-{{.max}})",
	MsgInvalidOp:   "Invalid selection, please enter 1-{{.max}}",
	MsgOpFailed:    "Operation failed",
	MsgOpSuccess:   "Operation successful",
	MsgLoading:     "Loading...",
	MsgFailed:      "Failed",
	MsgConfigSaved: "Configuration saved successfully!",
	MsgConfigPath:  "Config file location: {{.path}}",
	MsgReadyToUse:  "You can now get started:",

	MsgCmdExecFailed: "Command execution failed",
	MsgArgNotNumber:  "Argument must be a number",

	// Global flags
	MsgFlagTimeout:  "Request timeout in seconds",
	MsgFlagName:     "Resource name (alternative to --id)",
	MsgFlagOutput:   "Output format (table/json/csv/yaml)",
	MsgFlagProfile:  "Server profile name",
	MsgFlagAgent:    "Agent mode (disable interactive, JSON output, structured error codes)",
	MsgFlagDryRun:   "Dry-run mode (show what would be executed without actually running)",
	MsgFlagPage:     "Page number (starting from 1)",
	MsgFlagPageSize: "Items per page",
	MsgFlagIds:      "Multiple IDs (comma-separated)",

	// Agent commands
	MsgAgentCommandsShort: "List all available commands (machine-readable)",
	MsgAgentCommandsLong:  "Output all commands and their parameters in JSON format for programmatic agent usage",

	// Resource resolve errors
	MsgResolveNotFound:    "{{.type}} '{{.name}}' not found",
	MsgResolveMultiple:    "Multiple {{.type}} matched '{{.name}}', please use exact name or ID",
	MsgResolveUnsupported: "Unsupported resource type: {{.type}}",
	MsgResolveCannotParse: "Cannot parse ID from matched {{.type}}",

	MsgCfgHomeDirFailed:   "Failed to get user home directory",
	MsgCfgReadFailed:      "Failed to read config file",
	MsgCfgParseFailed:     "Failed to parse config file",
	MsgCfgSerializeFailed: "Failed to serialize config",
	MsgCfgWriteFailed:     "Failed to write config file",
	MsgCfgInvalidURL:      "Invalid server URL, must start with http:// or https://",

	// Root / Dashboard
	MsgRootShort: "Mayfly-Go CLI - Resource Connection & Management Tool",
	MsgRootLong: `Mayfly-Go CLI is a command-line tool for connecting to and managing various resources:
- Databases (MySQL, PostgreSQL, Oracle, etc.)
- Machines (SSH connections)
- Redis
- MongoDB

Agent usage:
  mayfly-cli --json login -s http://host -u admin -p password
  mayfly-cli --json db list
  mayfly-cli --json db exec --id 1 --db mydb --sql "SELECT 1"
  mayfly-cli --json ssh list
  mayfly-cli --json redis exec --id 1 --db 0 --cmd "GET key"

Use --json flag for structured JSON output, ideal for programmatic parsing.`,

	MsgDashboardTitle:      "🚀  Mayfly-Go CLI Resource Overview",
	MsgDashboardServer:     "Server:",
	MsgDashboardStatus:     "Status:",
	MsgDashboardLoggedIn:   "Logged in",
	MsgDashboardNotLogged:  "Not logged in - Please run the following command to login:",
	MsgDashboardAssets:     "Asset Statistics:",
	MsgDashboardType:       "Type",
	MsgDashboardIcon:       "Icon",
	MsgDashboardTotalCol:   "Total",
	MsgDashboardAvailCol:   "Available",
	MsgDashboardDesc:       "Description",
	MsgDashboardQuickStart: "💡 Quick Start:",
	MsgDashboardDb:         "Database:",
	MsgDashboardMachine:    "Machine:",
	MsgDashboardRedis:      "Redis:",
	MsgDashboardConfig:     "Config:",
	MsgDashboardOnlineFmt:  "{{.count}} online",
	MsgDashboardDbDesc:     "MySQL/PostgreSQL/SQLite and more",
	MsgDashboardRedisDesc:  "Standalone/Cluster/Sentinel",

	MsgLoginPromptTitle:  "⚠️  Not Logged In - Please Configure Connection",
	MsgLoginPromptServer: "Server Address:",
	MsgLoginPromptQuick:  "Quick Config:",
	MsgLoginPromptManual: "Or specify manually:",

	MsgAssetStatsFailed: "Failed to get asset statistics: {{.err}}",

	MsgDashboardDbName:      "Database",
	MsgDashboardMachineName: "Machine",
	MsgHelpersAuthRequired:  "Authentication failed, please login first",
	MsgTokenExpired:         "Login expired, please run mayfly-cli login to re-authenticate",

	// Logout / Whoami
	MsgLogoutShort:   "Logout and clear local credentials",
	MsgLogoutLong:    "Clear locally stored token and refreshToken",
	MsgLogoutSuccess: "Successfully logged out",
	MsgLogoutFailed:  "Logout failed",
	MsgWhoamiShort:   "Show current login status",
	MsgWhoamiLong:    "Display current server connection and authentication status",
	MsgWhoamiServer:  "Server",
	MsgWhoamiStatus:  "Status",
	MsgWhoamiUser:    "User",
	MsgWhoamiOnline:  "Logged in",
	MsgWhoamiOffline: "Not logged in",

	// Config / Login
	MsgConfigShort: "Configure CLI connection",
	MsgConfigLong: `Configure mayfly-go server address and login info.

Interactive mode (default):
  mayfly-cli config

Non-interactive mode (for agents):
  mayfly-cli config --server http://localhost:8888 --username admin --password admin123.
  mayfly-cli --json config -s http://localhost:8888 -u admin -p admin123.`,

	MsgConfigWizardTitle: "Mayfly-Go CLI First-time Setup Wizard",
	MsgConfigServerHint:  "Enter mayfly-go server address (e.g.: http://localhost:8889) [{{.server}}]: ",
	MsgConfigUserHint:    "Enter username (default: admin): ",
	MsgConfigPassHint:    "Enter password (or set MAYFLY_PASSWORD env var): ",

	MsgLoginShort: "Login to mayfly-go server and save token",
	MsgLoginLong: `Login to mayfly-go server to obtain and save access token.

Agent usage:
  mayfly-cli --json login -s http://host -u admin -p password
  mayfly-cli login --server http://host --username admin --password password`,

	MsgLoginLogging:    "Logging in...",
	MsgLoginStep1:      "[1/3] Getting RSA public key... ",
	MsgLoginStep2:      "[2/3] Encrypting password... ",
	MsgLoginStep3:      "[3/3] Logging in to get Token... ",
	MsgLoginSuccess:    "Login successful",
	MsgLoginFailed:     "Login failed",
	MsgLoginGetPubKey:  "Failed to get RSA public key: {{.err}}",
	MsgLoginEncrypt:    "Failed to encrypt password: {{.err}}",
	MsgLoginSaveCfg:    "Failed to save config: {{.err}}",
	MsgLoginNoPassword: "No password provided, use --password flag or MAYFLY_PASSWORD environment variable",

	MsgConfigFlagPath: "Config file path",
	MsgConfigFlagUser: "Username (skip interactive prompt if provided)",
	MsgConfigFlagPass: "Password (skip interactive prompt if provided)",
	MsgLoginFlagUser:  "Username",
	MsgLoginFlagPass:  "Password",

	// Database commands
	MsgDbShort:     "Database operations",
	MsgDbLong:      "Manage database connections and execute SQL queries",
	MsgDbListShort: "List available databases",
	MsgDbListLong: `List available database instances.

Agent usage:
  mayfly-cli --json db list
  mayfly-cli db list  # Interactive mode`,
	MsgDbListFailed:   "Failed to get database list: {{.err}}",
	MsgDbNoResource:   "No database resources found",
	MsgDbTableHeaders: "No.,Name,Type,Address",
	MsgDbTableTip:     "Enter number to select database, or q to quit",
	MsgDbExecShort:    "Execute SQL statement (using actual ID)",
	MsgDbExecLong: `Execute SQL statement using database actual ID, suitable for agents and scripts.

Examples:
  mayfly-cli --json db exec --id 1 --db mydb --sql "SELECT * FROM users LIMIT 10"
  mayfly-cli db exec --id 1 --db mydb --sql "SHOW TABLES"`,
	MsgDbExecRequired: "Must provide --id, --db, --sql parameters",
	MsgDbExecFailed:   "SQL execution failed",
	MsgDbSqlShort:     "Enter SQL execution mode",
	MsgDbSqlLong: `Enter SQL execution mode for a database.

Interactive mode:
  mayfly-cli db sql        # Interactive selection
  mayfly-cli db sql 1      # Select instance then interact

Non-interactive mode (agent):
  mayfly-cli --json db sql --id 1 --db mydb --sql "SELECT 1"
  mayfly-cli --json db sql --id 1 --db mydb  # Execute single SQL in specified db`,
	MsgDbSqlInvalidIdx: "Invalid database instance number: {{.arg}}, please enter 1-{{.max}}",
	MsgDbSqlGetDbInfo:  "🔍 Getting database info for instance {{.name}}...",
	MsgDbSqlNoDb:       "No databases available under this instance",
	MsgDbSqlSelectDb:   "📌 Select database (enter number)",
	MsgDbSqlDbList:     "📊 Database list for instance {{.name}}:",
	MsgDbSqlConnecting: "🔍 Verifying database connection...",
	MsgDbSqlConnected:  "✅ Database connected: {{.db}} ({{.type}})",
	MsgDbSqlConnFailed: "Database connection failed: {{.err}}",
	MsgDbTablesShort:   "View database table list",
	MsgDbTablesLong: `View table list for specified database.

Agent usage:
  mayfly-cli --json db tables --id 1 --db mydb`,
	MsgDbTablesRequired: "Must provide --id and --db parameters",
	MsgDbTablesFailed:   "Failed to get table list: {{.err}}",
	MsgDbDescShort:      "View table structure",
	MsgDbDescLong: `View table structure for specified table.

Agent usage:
  mayfly-cli --json db desc --id 1 --db mydb --table users`,
	MsgDbDescRequired: "Must provide --id, --db, --table parameters",
	MsgDbDescFailed:   "Failed to query table structure: {{.err}}",

	MsgDbColumnsShort: "View table column metadata (with comments)",
	MsgDbColumnsLong: `View table column metadata including column name, type, primary key, default value, comments, etc.

Examples:
  mayfly-cli db columns --id 1 --db mydb --table users
  mayfly-cli --json db columns --id 1 --db mydb --table users`,
	MsgDbColumnsRequired: "--id, --db, --table are required",
	MsgDbColumnsFailed:   "Failed to query column metadata: {{.err}}",

	MsgDbDdlShort: "View CREATE TABLE statement",
	MsgDbDdlLong: `View the CREATE TABLE DDL statement for a table.

Examples:
  mayfly-cli db ddl --id 1 --db mydb --table users
  mayfly-cli --json db ddl --id 1 --db mydb --table users`,
	MsgDbDdlRequired: "--id, --db, --table are required",
	MsgDbDdlFailed:   "Failed to query DDL: {{.err}}",

	MsgDbIndexShort: "View table index info",
	MsgDbIndexLong: `View table index information.

Examples:
  mayfly-cli db index --id 1 --db mydb --table users
  mayfly-cli --json db index --id 1 --db mydb --table users`,
	MsgDbIndexRequired: "--id, --db, --table are required",
	MsgDbIndexFailed:   "Failed to query index info: {{.err}}",

	MsgDbDumpShort: "Export database SQL",
	MsgDbDumpLong: `Export database structure or data as SQL file.

type: 1=structure, 2=data, 3=structure+data

Examples:
  mayfly-cli db dump --id 1 --db mydb --type 3
  mayfly-cli db dump --id 1 --db mydb --type 1 --tables users,orders -o ./backup.sql`,
	MsgDbDumpRequired: "--id, --db, --type are required",
	MsgDbDumpFailed:   "Export failed: {{.err}}",
	MsgDbDumpOk:       "✅ Exported to",

	MsgDbMenuTitle:         "📊  Database List",
	MsgDbMenuGetInfo:       "Get database info",
	MsgDbMenuExec:          "Execute SQL",
	MsgDbMenuTables:        "View table list",
	MsgDbMenuDesc:          "View table structure",
	MsgDbMenuBack:          "Back to database list",
	MsgDbMenuExit:          "Exit",
	MsgDbMenuSelect:        "📌 Select database instance (enter number, q to quit): ",
	MsgDbMenuInvalid:       "❌ Failed to get db info: {{.err}}",
	MsgDbMenuGetInfoFailed: "Failed to get database info",
	MsgDbMenuNoDb:          "⚠️  No databases available under this instance",
	MsgDbMenuSelectDb:      "📌 Select database (enter number, b to go back, q to quit): ",
	MsgDbMenuConnTest:      "🔍 Verifying database connection...",
	MsgDbMenuConnFail:      "❌ Database connection failed: {{.err}}",
	MsgDbMenuConnOk:        "✅ Database connected: {{.db}} ({{.type}})",
	MsgDbMenuRetry:         "Please select another database or press q to quit",

	MsgDbSqlPrompt:        "💻 Enter SQL (type 'back' to go back, 'quit' to exit):",
	MsgDbSqlHint1:         "Tip: SQL statements supported, e.g. SELECT * FROM table_name LIMIT 10",
	MsgDbSqlHint2:         "      Type 'history' to view history, 'clear' to clear history",
	MsgDbSqlHint3:         "      Type 'help' for SQL examples, 'tables' to view table list",
	MsgDbSqlHint4:         "      Type 'desc <table_name>' to view table structure",
	MsgDbSqlGetTables:     "🔍 Getting table list...",
	MsgDbSqlGetInfo:       "🔍 Getting database info...",
	MsgDbSqlInfoTitle:     "📊 Database Info:",
	MsgDbSqlInfoFailed:    "❌ Failed to get database info: {{.err}}",
	MsgDbSqlDescSpecify:   "❌ Please specify table name, e.g.: desc table_name",
	MsgDbSqlDescUnsupport: "⚠️  Table structure query not supported for {{.type}} database",
	MsgDbSqlQueryFailed:   "❌ Query failed: {{.err}}",

	MsgDbHistoryNone:    "📝 No history",
	MsgDbHistoryTitle:   "📝 SQL History (last {{.count}} entries):",
	MsgDbHistoryCleared: "🗑️  History cleared",

	MsgDbHelpTitle:       "📚 Common SQL Examples:",
	MsgDbHelpQuery:       "📌 Query Operations:",
	MsgDbHelpStruct:      "📌 Table Structure Operations:",
	MsgDbHelpShortcut:    "📌 Quick Commands:",
	MsgDbHelpAgent:       "💡 Agent Direct Call:",
	MsgDbHelpAgentExec:   "mayfly-cli --json db exec --id <ID> --db <db_name> --sql \"<SQL>\"",
	MsgDbHelpAgentTables: "mayfly-cli --json db tables --id <ID> --db <db_name>",
	MsgDbHelpAgentDesc:   "mayfly-cli --json db desc --id <ID> --db <db_name> --table <table_name>",

	MsgDbErrTitle:        "❌ SQL Execution Failed",
	MsgDbErrConnRefused:  "🔌 Connection refused: Cannot connect to database server",
	MsgDbErrNoDb:         "🗄️  Database does not exist: Invalid database name",
	MsgDbErrAccessDenied: "🔒 Access denied: Invalid username or password",
	MsgDbErrSyntax:       "📝 SQL syntax error: Incorrect SQL statement format",
	MsgDbErrNoTable:      "📋 Table does not exist: Invalid table name",
	MsgDbErrTimeout:      "⏰ Execution timeout: SQL query took too long",
	MsgDbErrDetail:       "❌ Error details: {{.err}}",

	MsgDbQueryOk:         "✅ Query successful, no results returned",
	MsgDbQueryRecords:    "✅ {{.count}} records total",
	MsgDbQueryShowFirst:  "... ({{.count}} records total, showing first 100)",
	MsgDbGetDbListFailed: "Failed to get database list: {{.err}}",
	MsgDbConnFailed:      "Connection failed: {{.err}}",
	MsgDbConnAnomaly:     "Connection anomaly: No return value",
	MsgDbSelectIdx:       "📌 Select database (enter number, b to go back, q to quit): ",

	MsgDbSqlUse:                    "sql [index]",
	MsgDbSqlGetListFail:            "Failed to get database list",
	MsgDbSqlNoRes:                  "No database resources found",
	MsgDbSqlGetInfoFail:            "Failed to get database info",
	MsgDbSqlInvalidDbIdx:           "Invalid database instance number: {{.arg}}, please enter 1-{{.max}}",
	MsgDbSqlConnFail:               "Database connection failed",
	MsgDbNonInteractiveGetListFail: "Failed to get database list",
	MsgDbNonInteractiveConnFail:    "Database connection failed",
	MsgDbNonInteractiveExecFail:    "Failed to execute SQL",
	MsgDbNonInteractiveConnOk:      "✅ Database connected: {{.db}} (id={{.id}})",
	MsgDbSqlResultOk:               "✅ Query successful, no results returned",
	MsgDbFlagExecId:                "Database instance ID (from db list)",
	MsgDbFlagExecDb:                "Database name",
	MsgDbFlagExecSql:               "SQL statement to execute",
	MsgDbFlagSqlId:                 "Database instance ID",
	MsgDbFlagSqlDb:                 "Database name",
	MsgDbFlagSqlSql:                "SQL statement to execute (optional, test connection only if not provided)",
	MsgDbFlagTablesId:              "Database instance ID",
	MsgDbFlagTablesDb:              "Database name",
	MsgDbFlagDescId:                "Database instance ID",
	MsgDbFlagDescDb:                "Database name",
	MsgDbFlagDescTable:             "Table name",
	MsgDbFlagDescType:              "Database type (mysql/postgresql/sqlite/dm)",
	MsgDbFlagColumnsId:             "Database instance ID (from db list)",
	MsgDbFlagColumnsDb:             "Database name",
	MsgDbFlagColumnsTable:          "Table name",
	MsgDbFlagDdlId:                 "Database instance ID (from db list)",
	MsgDbFlagDdlDb:                 "Database name",
	MsgDbFlagDdlTable:              "Table name",
	MsgDbFlagIndexId:               "Database instance ID (from db list)",
	MsgDbFlagIndexDb:               "Database name",
	MsgDbFlagIndexTable:            "Table name",
	MsgDbFlagDumpId:                "Database instance ID (from db list)",
	MsgDbFlagDumpDb:                "Database name",
	MsgDbFlagDumpType:              "Export type: 1=structure, 2=data, 3=structure+data",
	MsgDbFlagDumpTables:            "Table names (comma-separated, optional)",
	MsgDbFlagDumpOutput:            "Output file path (default: stdout)",
	MsgDbHelpTables:                "tables                    -- View all tables",
	MsgDbHelpDesc:                  "desc table_name           -- View table structure",
	MsgDbHelpHelp:                  "help     -- Show this help message",
	MsgDbHelpHistory:               "history  -- View SQL history",
	MsgDbHelpTablesList:            "tables   -- View table list",
	MsgDbHelpDescTable:             "desc <table_name> -- View table structure",
	MsgDbHelpBack:                  "back     -- Back to database list",
	MsgDbHelpQuit:                  "quit     -- Exit program",

	// SSH commands
	MsgSshShort:     "SSH connection management",
	MsgSshLong:      "Manage machine SSH connections and execute remote commands",
	MsgSshListShort: "List available machines",
	MsgSshListLong: `List available machine instances.

Agent usage:
  mayfly-cli --json ssh list
  mayfly-cli ssh list  # Interactive mode`,
	MsgSshListFailed:   "Failed to get machine list: {{.err}}",
	MsgSshNoResource:   "No machine resources found",
	MsgSshTableHeaders: "No.,Name,Address,Status",
	MsgSshTableTip:     "Enter number to select machine, or q to quit",
	MsgSshStatsShort:   "View machine statistics",
	MsgSshStatsLong: `View CPU/memory/disk statistics for specified machine.

Agent usage:
  mayfly-cli --json ssh stats --id 1`,
	MsgSshStatsRequired: "Must provide --id parameter",
	MsgSshStatsFailed:   "Failed to get statistics: {{.err}}",
	MsgSshStatsTitle:    "📊 Machine Statistics (ID: {{.id}})",
	MsgSshProcessShort:  "View machine process list",
	MsgSshProcessLong: `View process list for specified machine.

Agent usage:
  mayfly-cli --json ssh process --id 1`,
	MsgSshProcessRequired: "Must provide --id parameter",
	MsgSshProcessFailed:   "Failed to get process list: {{.err}}",
	MsgSshProcessTitle:    "📋 Process List (ID: {{.id}})",
	MsgSshUsersShort:      "View machine user list",
	MsgSshUsersLong: `View user list for specified machine.

Agent usage:
  mayfly-cli --json ssh users --id 1`,
	MsgSshUsersRequired: "Must provide --id parameter",
	MsgSshUsersFailed:   "Failed to get user list: {{.err}}",
	MsgSshUsersTitle:    "👥 User List (ID: {{.id}})",
	MsgSshUsersNoData:   "No user data",
	MsgSshUsersCount:    "{{.count}} users total",
	MsgSshExecShort:     "Execute command on remote machine",
	MsgSshExecLong: `Execute commands non-interactively on remote machine and return results.

Agent usage:
  mayfly-cli --json ssh exec --id 1 --cmd "ls -la"
  mayfly-cli --json ssh exec --id 1 --cmd "df -h" --auth-cert certName`,
	MsgSshExecRequired:    "Must provide --id parameter",
	MsgSshExecCmdRequired: "Must provide --cmd parameter",
	MsgSshExecNotFound:    "Machine ID={{.id}} not found or cannot get auth credential",
	MsgSshExecFailed:      "Command execution failed: {{.err}}",
	MsgSshExecTitle:       "📋 Command Result (Machine ID: {{.id}})",
	MsgSshExecErrLabel:    "⚠️  Execution error: {{.err}}",
	MsgSshConnectShort:    "Connect to SSH terminal",
	MsgSshConnectLong: `Connect to remote machine SSH terminal via WebSocket.

Agent usage:
  mayfly-cli ssh connect --id 1
  mayfly-cli ssh connect --id 1 --auth-cert certName`,
	MsgSshConnectRequired: "Must provide --id parameter",
	MsgSshConnectNotFound: "Machine ID={{.id}} not found or cannot get auth credential",
	MsgSshConnecting:      "🚀 Connecting to SSH terminal (Machine ID: {{.id}})...",
	MsgSshConnectFailed:   "SSH terminal connection failed: {{.err}}",
	MsgSshConnectClosed:   "✅ SSH session closed",

	MsgSshMenuTitle:    "🛠️   {{.name}} Operation Menu",
	MsgSshMenuTerminal: "SSH Terminal",
	MsgSshMenuSysInfo:  "View system info",
	MsgSshMenuProcess:  "View process list",
	MsgSshMenuUsers:    "View user list",
	MsgSshMenuBack:     "Back to machine list",
	MsgSshMenuExit:     "Exit",
	MsgSshMenuSelect:   "📌 Select machine (enter number, q to quit): ",
	MsgSshMenuInvalid:  "❌ Invalid number, please enter 1-{{.max}}",
	MsgSshSelected:     "✅ Selected: {{.name}}",
	MsgSshOffline:      "⚠️  Machine [{{.name}}] is offline, cannot operate",

	MsgSshProcessHeader: "PID,User,CPU%,MEM%,Command",
	MsgSshProcessNoData: "No process data",
	MsgSshProcessMore:   "... ({{.count}} lines total, showing first 25)",
	MsgSshProcessTotal:  "... ({{.count}} processes total)",

	MsgSshUsersHeader: "Username,Groups,Shell",
	MsgSshNoCert:      "⚠️  Cannot get auth credential info",

	MsgSshFlagStatsId:     "Machine ID (from ssh list)",
	MsgSshFlagProcessId:   "Machine ID",
	MsgSshFlagUsersId:     "Machine ID",
	MsgSshFlagExecId:      "Machine ID",
	MsgSshFlagExecCmd:     "Command to execute",
	MsgSshFlagExecCert:    "Auth credential name (optional, auto-detected if not specified)",
	MsgSshFlagConnectId:   "Machine ID",
	MsgSshFlagConnectCert: "Auth credential name (optional, auto-detected if not specified)",

	MsgSshSysInfoTitle:   "📊  {{.name}} System Info",
	MsgSshSysInfoGetFail: "❌ Failed to get: {{.err}}",
	MsgSshSysInfoCpu:     "CPU Usage:",
	MsgSshSysInfoMem:     "Memory Usage:",
	MsgSshSysInfoDisk:    "Disk Usage:",
	MsgSshSysInfoLoad:    "System Load (1min):",

	MsgSshProcessGetFail:   "❌ Failed to get: {{.err}}",
	MsgSshProcessNoData2:   "No process data",
	MsgSshUserListTitle:    "👥  {{.name}} User List",
	MsgSshUserListGetFail:  "❌ Failed to get: {{.err}}",
	MsgSshUserListNoData:   "No user data",
	MsgSshUserListHeader:   "Username,Groups,Shell",
	MsgSshUserListCount:    "{{.count}} users total",
	MsgSshTerminalConnFail: "❌ Terminal connection failed: {{.err}}",
	MsgSshTerminalClosed:   "✅ Terminal session closed",

	// SSH file operations
	MsgSshLsShort: "List remote directory contents",
	MsgSshLsLong: `List files and subdirectories in a remote machine directory.

Examples:
  mayfly-cli ssh ls --id 1 --path /etc
  mayfly-cli --json ssh ls --id 1 --path /var/log`,
	MsgSshLsRequired: "--id, --path are required",
	MsgSshLsFailed:   "Failed to list directory: {{.err}}",
	MsgSshCatShort:   "Read remote file content",
	MsgSshCatLong: `Read file content from a remote machine (file must be < 1MB).

Examples:
  mayfly-cli ssh cat --id 1 --path /etc/hosts
  mayfly-cli --json ssh cat --id 1 --path /etc/nginx/nginx.conf`,
	MsgSshCatRequired:   "--id, --path are required",
	MsgSshCatFailed:     "Failed to read file: {{.err}}",
	MsgSshDownloadShort: "Download remote file to local",
	MsgSshDownloadLong: `Download a file from a remote machine to local.

Examples:
  mayfly-cli ssh download --id 1 --path /var/log/app.log
  mayfly-cli ssh download --id 1 --path /etc/nginx/nginx.conf -o ./conf/`,
	MsgSshDownloadRequired:    "--id, --path are required",
	MsgSshDownloadFailed:      "Download failed: {{.err}}",
	MsgSshDownloadWriteFailed: "Failed to write local file: {{.err}}",
	MsgSshDownloadOk:          "✅ Downloaded",
	MsgSshUploadShort:         "Upload local file to remote machine",
	MsgSshUploadLong: `Upload a local file to a remote machine directory.

Examples:
  mayfly-cli ssh upload --id 1 --path /tmp/ --file ./deploy.sh
  mayfly-cli --json ssh upload --id 1 --path /opt/app/ --file ./app.jar`,
	MsgSshUploadRequired:   "--id, --path, --file are required",
	MsgSshUploadReadFailed: "Failed to read local file: {{.err}}",
	MsgSshUploadFailed:     "Upload failed: {{.err}}",
	MsgSshUploadOk:         "✅ Uploaded",
	MsgSshWriteShort:       "Write content to remote file",
	MsgSshWriteLong: `Write content to a file on a remote machine.

Examples:
  mayfly-cli ssh write --id 1 --path /tmp/test.txt --content "hello world"`,
	MsgSshWriteRequired: "--id, --path, --content are required",
	MsgSshWriteFailed:   "Write failed: {{.err}}",
	MsgSshWriteOk:       "✅ Written",
	MsgSshRmShort:       "Remove remote file/directory",
	MsgSshRmLong: `Remove a file or directory on a remote machine.

Examples:
  mayfly-cli ssh rm --id 1 --path /tmp/test.txt`,
	MsgSshRmRequired: "--id, --path are required",
	MsgSshRmFailed:   "Remove failed: {{.err}}",
	MsgSshRmOk:       "✅ Removed",
	MsgSshKillShort:  "Kill remote process",
	MsgSshKillLong: `Kill a process on a remote machine (kill -9).

Examples:
  mayfly-cli ssh kill --id 1 --pid 12345`,
	MsgSshKillRequired: "--id, --pid are required",
	MsgSshKillFailed:   "Failed to kill process: {{.err}}",
	MsgSshKillOk:       "✅ Killed",

	MsgSshFlagId:        "Machine ID (from ssh list)",
	MsgSshFlagPath:      "Remote file/directory path",
	MsgSshFlagAuthCert:  "Auth credential name (optional, auto-detected)",
	MsgSshFlagOutput:    "Local save path (default: current directory)",
	MsgSshFlagRemoteDir: "Remote target directory",
	MsgSshFlagLocalFile: "Local file path",
	MsgSshFlagContent:   "Content to write",
	MsgSshFlagPid:       "Process PID to kill",
	MsgSshFlagSrc:       "Source path",
	MsgSshFlagDst:       "Destination path",
	MsgSshFlagName:      "New file name",

	// SSH file operations
	MsgSshCpShort:     "Copy remote file or directory",
	MsgSshCpLong:      "Copy a file or directory on the remote machine",
	MsgSshCpOk:        "Copied",
	MsgSshMvShort:     "Move/rename remote file or directory",
	MsgSshMvLong:      "Move or rename a file or directory on the remote machine",
	MsgSshMvOk:        "Moved",
	MsgSshRenameShort: "Rename a remote file (keep same directory)",
	MsgSshRenameOk:    "Renamed",
	MsgSshMkdirShort:  "Create remote directory",
	MsgSshMkdirLong:   "Create a directory on the remote machine",
	MsgSshMkdirOk:     "Created directory",
	MsgSshTouchShort:  "Create an empty remote file",
	MsgSshTouchOk:     "Created file",
	MsgSshStatShort:   "View file/directory status info",
	MsgSshOpFailed:    "Operation failed",
	MsgSshOpRequired:  "Missing required arguments",

	// Redis commands
	MsgRedisShort:     "Redis operations",
	MsgRedisLong:      "Manage Redis connections and execute commands",
	MsgRedisListShort: "List available Redis instances",
	MsgRedisListLong: `List available Redis instances.

Agent usage:
  mayfly-cli --json redis list`,
	MsgRedisListFailed:   "Failed to get Redis list: {{.err}}",
	MsgRedisNoResource:   "No Redis resources found",
	MsgRedisTableHeaders: "No.,Name,Address,Mode",
	MsgRedisTableTip:     "Enter number to select Redis, or q to quit",
	MsgRedisExecShort:    "Execute Redis command (using actual ID)",
	MsgRedisExecLong: `Execute Redis command using actual ID, suitable for agents and scripts.

Examples:
  mayfly-cli --json redis exec --id 1 --db 0 --cmd "GET mykey"
  mayfly-cli --json redis exec --id 1 --db 0 --cmd "SET foo bar"
  mayfly-cli --json redis exec --id 1 --db 0 --cmd "KEYS *"`,
	MsgRedisExecRequired: "Must provide --id and --cmd parameters",
	MsgRedisExecFailed:   "Failed to execute Redis command: {{.err}}",
	MsgRedisInfoShort:    "View Redis server info",
	MsgRedisInfoLong: `View Redis server detailed info.

Agent usage:
  mayfly-cli --json redis info --id 1`,
	MsgRedisInfoRequired: "Must provide --id parameter",
	MsgRedisInfoFailed:   "Query failed: {{.err}}",
	MsgRedisInfoTitle:    "📊 Redis Server Info (ID: {{.id}})",
	MsgRedisMemShort:     "View Redis memory usage",
	MsgRedisMemLong: `View Redis memory usage info.

Agent usage:
  mayfly-cli --json redis mem --id 1`,
	MsgRedisMemRequired: "Must provide --id parameter",
	MsgRedisMemFailed:   "Query failed: {{.err}}",
	MsgRedisMemTitle:    "💾 Redis Memory Usage (ID: {{.id}})",

	MsgRedisMenuTitle:    "🔴  {{.name}} Redis Operation Menu",
	MsgRedisMenuExec:     "Execute Redis command",
	MsgRedisMenuInfo:     "View server info",
	MsgRedisMenuMem:      "View memory usage",
	MsgRedisMenuKeyspace: "View keyspace statistics",
	MsgRedisMenuBack:     "Back to Redis list",
	MsgRedisMenuExit:     "Exit",
	MsgRedisMenuSelect:   "📌 Select Redis (enter number, q to quit): ",
	MsgRedisMenuInvalid:  "❌ Invalid number, please enter 1-{{.max}}",
	MsgRedisSelected:     "✅ Selected: {{.name}} ({{.mode}})",

	MsgRedisExecPrompt: "💻 Enter Redis command (type 'back' to go back, 'quit' to exit):",
	MsgRedisExecHint:   "Tip: Format is 'DB# command', e.g. '0 KEYS *'",
	MsgRedisExecFmtErr: "❌ Format error, please use: DB# command",
	MsgRedisExecDbErr:  "❌ Invalid DB number: {{.db}}",
	MsgRedisExecFail:   "❌ Execution failed: {{.err}}",

	MsgRedisInfoVersion:   "Version:",
	MsgRedisInfoOs:        "OS:",
	MsgRedisInfoPort:      "Port:",
	MsgRedisInfoPid:       "Process ID:",
	MsgRedisInfoUptime:    "Uptime:",
	MsgRedisInfoQueryFail: "❌ Query failed: {{.err}}",

	MsgRedisMemUsed:      "Used:",
	MsgRedisMemMax:       "Max:",
	MsgRedisMemPeak:      "Peak:",
	MsgRedisMemRss:       "RSS:",
	MsgRedisMemQueryFail: "❌ Query failed: {{.err}}",

	MsgRedisKeyspaceTitle:     "🔑  {{.name}} Keyspace Statistics",
	MsgRedisKeyspaceNoData:    "No keyspace data",
	MsgRedisKeyspaceHeader:    "Database,Keys,Expires",
	MsgRedisKeyspaceQueryFail: "❌ Query failed: {{.err}}",

	MsgRedisFlagExecId:  "Redis instance ID (from redis list)",
	MsgRedisFlagExecDb:  "Redis DB number",
	MsgRedisFlagExecCmd: "Redis command to execute",
	MsgRedisFlagInfoId:  "Redis instance ID",
	MsgRedisFlagMemId:   "Redis instance ID",

	MsgRedisScanShort: "Scan Redis keys",
	MsgRedisScanLong: `Scan Redis keys using SCAN command with pattern matching.

Examples:
  mayfly-cli redis scan --id 1 --db 0 --match "user:*" --count 100
  mayfly-cli --json redis scan --id 1 --db 0`,
	MsgRedisScanRequired:  "--id is required",
	MsgRedisScanFailed:    "Failed to scan keys: {{.err}}",
	MsgRedisFlagScanId:    "Redis instance ID (from redis list)",
	MsgRedisFlagScanDb:    "Redis DB number",
	MsgRedisFlagScanMatch: "Key match pattern (default: *)",
	MsgRedisFlagScanCount: "Scan count per iteration (default: 100)",

	// Redis Key operations
	MsgRedisKeyInfoShort: "View key type, value and TTL",
	MsgRedisKeyInfoLong:  "Inspect a Redis key's type, value, and TTL",
	MsgRedisKeyTtlShort:  "View key TTL (seconds)",
	MsgRedisKeyMemShort:  "View key memory usage (bytes)",
	MsgRedisKeyTtlResult: "TTL: {{.ttl}}",
	MsgRedisKeyMemResult: "Memory: {{.mem}} bytes",
	MsgRedisFlagKeyId:    "Redis instance ID",
	MsgRedisFlagKeyDb:    "Redis database number",
	MsgRedisFlagKeyName:  "Redis key name",

	// Terminal common
	MsgTermTip:          "💡 Tip:",
	MsgTermNoResource:   "No available {{.type}} resources",
	MsgTermSelectRes:    "Select {{.type}} number (1-{{.max}}): ",
	MsgTermInputInvalid: "Invalid input: {{.err}}",
	MsgTermOutOfRange:   "Number out of range (1-{{.max}})",

	// Client internal errors
	MsgClientSerializeFailed:    "Failed to serialize request body",
	MsgClientCreateReqFailed:    "Failed to create request",
	MsgClientSendReqFailed:      "Failed to send request",
	MsgClientReadRespFailed:     "Failed to read response",
	MsgClientRequestFailed:      "Request failed",
	MsgClientParseRespFailed:    "Failed to parse response data",
	MsgClientEncryptSqlFailed:   "Failed to encrypt SQL",
	MsgClientConnFailed:         "Connection failed",
	MsgClientConnAnomaly:        "Connection anomaly: No return value",
	MsgClientGetInstDbsFailed:   "Failed to get instance databases",
	MsgClientGetDbInstFailed:    "Failed to get database instance info",
	MsgClientNoCertInfo:         "Auth credential info not found",
	MsgClientGetTableFailed:     "Failed to get table info",
	MsgClientGetDbInfoFailed:    "Failed to get database info",
	MsgClientAesKeyTooShort:     "AES key too short: need 24 bytes, got {{.len}}",
	MsgClientAesCipherFailed:    "Failed to create AES cipher",
	MsgClientTerminalConnFailed: "Failed to connect terminal",
	MsgClientTerminalModeFailed: "Failed to set terminal mode",
	MsgClientHttpFailed:         "HTTP request failed",
	MsgClientHttpStatus:         "HTTP status code: {{.code}}",
	MsgClientReadRespFailed2:    "Failed to read response",
	MsgClientParseRespFailed2:   "Failed to parse response",
	MsgClientServerError:        "Server error: {{.msg}}",
	MsgClientNoPublicKey:        "Public key not found in response",
	MsgClientParsePEMFailed:     "Failed to parse PEM format public key",
	MsgClientParseKeyFailed:     "Failed to parse public key",
	MsgClientNotRSAKey:          "Not an RSA public key",
	MsgClientEncryptFailed:      "Encryption failed",
	MsgClientSerializeFailed2:   "Failed to serialize request",
	MsgClientHttpReqFailed:      "HTTP request failed",
	MsgClientReadRespFailed3:    "Failed to read response",
	MsgClientParseRespFailed3:   "Failed to parse response",
	MsgClientLoginFailed:        "Login failed: {{.msg}}",
	MsgClientRespFormatErr:      "Response data format error",
	MsgClientNoToken:            "Token field not found",

	// MongoDB
	MsgMongoShort:          "MongoDB operations",
	MsgMongoLong:           "Manage MongoDB connections, browse databases/collections, and run queries",
	MsgMongoListShort:      "List available MongoDB instances",
	MsgMongoNoResource:     "No MongoDB instances found",
	MsgMongoDbShort:        "List databases in a MongoDB instance",
	MsgMongoColsShort:      "List collections in a database",
	MsgMongoFindShort:      "Query documents in a collection",
	MsgMongoFindLong:       "Query documents using JSON filter",
	MsgMongoRunShort:       "Run a MongoDB command (JSON)",
	MsgMongoRunLong:        "Execute a raw MongoDB command",
	MsgMongoTableTip:       "Use: mayfly-cli mongo databases --id <ID>",
	MsgMongoFlagId:         "MongoDB instance ID",
	MsgMongoFlagDb:         "Database name",
	MsgMongoFlagCollection: "Collection name",
	MsgMongoFlagFilter:     "Query filter (JSON)",
	MsgMongoFlagLimit:      "Max documents to return",
	MsgMongoFlagSkip:       "Documents to skip",
	MsgMongoFlagCommand:    "MongoDB command (JSON)",
	MsgMongoInvalidFilter:  "Invalid --filter JSON: {{.err}}",
	MsgMongoInvalidCommand: "Invalid --command JSON: {{.err}}",

	// Ping
	MsgPingShort:    "Check server connectivity and auth status",
	MsgPingLong:     "Verify mayfly-go server is reachable and token is valid",
	MsgPingFailed:   "Ping failed: {{.err}}",
	MsgPingTitle:    "Mayfly-Go Server Status",
	MsgPingServer:   "Server",
	MsgPingAuth:     "Auth",
	MsgPingLatency:  "Latency",
	MsgPingAuthOk:   "authenticated",
	MsgPingAuthAnon: "anonymous",

	// DB Version
	MsgDbVersionShort:  "View database server version",
	MsgDbVersionLong:   "Get the version info of a database instance",
	MsgDbVersionResult: "Database ID {{.id}} ({{.db}}) version: {{.version}}",
	MsgDbFlagVersionId: "Database instance ID",
	MsgDbFlagVersionDb: "Database name",
}
