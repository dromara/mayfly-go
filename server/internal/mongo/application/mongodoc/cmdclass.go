package mongodoc

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// 数据面权限码。分级鉴权与路由注册共用这一处定义，避免「路由声明一套、命令分级另一套」造成漂移。
const (
	// PermDataSave 文档写入（insert/update/findAndModify/聚合写出）。
	PermDataSave = "mongo:data:save"
	// PermDataDel 文档删除。
	PermDataDel = "mongo:data:del"
	// PermDDLSave 结构变更（建集合、建索引、改集合选项）。
	PermDDLSave = "mongo:ddl:save"
	// PermDDLDel 结构销毁（删集合、删库、删索引）。
	PermDDLDel = "mongo:ddl:del"
	// PermCmdAdmin 服务器管理（账号、角色、副本集、分片、进程与参数）。
	PermCmdAdmin = "mongo:cmd:admin"
)

// Level 命令的语义级别，是「要求哪个权限码」与「前端是否需要二次确认」的唯一依据。
//
// 级别按严重度单调递增，因此「多条规则命中同一命令」只需取最大值；未知命令一律落到最高级。
// 之所以要区分文档级与结构级：删光一个集合的文档与删掉这个集合本身，
// 前者可靠应用层条件约束，后者会连带销毁索引与集合选项，运维上属于不同的授权决策。
type Level int8

const (
	LevelRead       Level = iota // 只读，仅受资源可见性约束
	LevelDataSave                // 写文档
	LevelDataDel                 // 删文档
	LevelStructSave              // 建/改结构
	LevelStructDel               // 销毁结构
	LevelAdmin                   // 服务器管理
)

func (l Level) String() string {
	switch l {
	case LevelRead:
		return "read"
	case LevelDataSave:
		return "dataSave"
	case LevelDataDel:
		return "dataDel"
	case LevelStructSave:
		return "structSave"
	case LevelStructDel:
		return "structDel"
	default:
		return "admin"
	}
}

// PermissionCode 该级别要求的权限码；LevelRead 返回空串，表示只受资源可见性约束。
func (l Level) PermissionCode() string {
	switch l {
	case LevelDataSave:
		return PermDataSave
	case LevelDataDel:
		return PermDataDel
	case LevelStructSave:
		return PermDDLSave
	case LevelStructDel:
		return PermDDLDel
	case LevelAdmin:
		return PermCmdAdmin
	default:
		return ""
	}
}

// NeedConfirm 执行前是否需要二次确认：会销毁数据或越出当前集合范围的一律需要。
//
// 只读与常规写入不需要确认，与 redis 模块「服务端标记优先、前端不再自立名单」的口径一致。
func (l Level) NeedConfirm() bool {
	return l >= LevelDataDel
}

var (
	// ErrEmptyCommand 命令文档为空，无法判定命令名。
	ErrEmptyCommand = errors.New("mongodoc: command document is empty")
)

// commandLevels 命令名 → 语义级别。
//
// 命令名精确匹配，不做大小写归一：Mongo 命令名本就大小写敏感，`Drop` 这类写法服务端会报
// unknown command；这里让异常写法一律落到「未登记 → LevelAdmin」的 fail-closed 分支，
// 而不是靠归一把它救活成可执行命令。
var commandLevels = map[string]Level{
	// ---- 只读 ----
	"ping":                LevelRead,
	"hello":               LevelRead,
	"isMaster":            LevelRead,
	"whatsmyuri":          LevelRead,
	"buildInfo":           LevelRead,
	"getCmdLineArgument":  LevelRead,
	"hostInfo":            LevelRead,
	"serverStatus":        LevelRead,
	"connPoolStats":       LevelRead,
	"getLog":              LevelRead,
	"top":                 LevelRead,
	"fcv":                 LevelRead,
	"getParameter":        LevelRead,
	"checkFreeMonitoring": LevelRead,
	"listDatabases":       LevelRead,
	"listCollections":     LevelRead,
	"dbStats":             LevelRead,
	"collStats":           LevelRead,
	"listIndexes":         LevelRead,
	"indexStats":          LevelRead,
	"planCacheListPlans":  LevelRead,
	"planCacheStats":      LevelRead,
	"validate":            LevelRead,
	"currentOp":           LevelRead,
	"profile":             LevelRead,
	"find":                LevelRead,
	"aggregate":           LevelRead,
	"count":               LevelRead,
	"distinct":            LevelRead,
	"getMore":             LevelRead,
	"killCursors":         LevelRead,
	"explain":             LevelRead,
	"usersInfo":           LevelRead,
	"rolesInfo":           LevelRead,
	"replSetGetStatus":    LevelRead,
	"saslStart":           LevelRead,
	"saslContinue":        LevelRead,

	// ---- 写文档 ----
	"insert":        LevelDataSave,
	"update":        LevelDataSave,
	"findAndModify": LevelDataSave,
	"bulkWrite":     LevelDataSave,
	"mapReduce":     LevelDataSave,
	"group":         LevelDataSave,

	// ---- 删文档 ----
	"delete": LevelDataDel,

	// ---- 建/改结构 ----
	"create":           LevelStructSave,
	"createIndexes":    LevelStructSave,
	"collMod":          LevelStructSave,
	"reIndex":          LevelStructSave,
	"renameCollection": LevelStructSave,
	"view":             LevelStructSave,

	// ---- 销毁结构 ----
	"drop":         LevelStructDel,
	"dropDatabase": LevelStructDel,
	"dropIndexes":  LevelStructDel,

	// ---- 服务器管理 ----
	"createUser":                     LevelAdmin,
	"updateUser":                     LevelAdmin,
	"dropUser":                       LevelAdmin,
	"changePassword":                 LevelAdmin,
	"grantRolesToUser":               LevelAdmin,
	"revokeRolesFromUser":            LevelAdmin,
	"dropAllUsersFromDatabase":       LevelAdmin,
	"createRole":                     LevelAdmin,
	"updateRole":                     LevelAdmin,
	"dropRole":                       LevelAdmin,
	"dropAllRolesFromDatabase":       LevelAdmin,
	"grantPrivilegesToRole":          LevelAdmin,
	"revokePrivilegesFromRole":       LevelAdmin,
	"setParameter":                   LevelAdmin,
	"shutdown":                       LevelAdmin,
	"fsync":                          LevelAdmin,
	"fsyncUnlock":                    LevelAdmin,
	"compact":                        LevelAdmin,
	"killOp":                         LevelAdmin,
	"configureFailPoint":             LevelAdmin,
	"setFeatureCompatibilityVersion": LevelAdmin,
	"setClusterParameter":            LevelAdmin,
	"replSetInitiate":                LevelAdmin,
	"replSetReconfig":                LevelAdmin,
	"replSetStepDown":                LevelAdmin,
	"replSetStepUp":                  LevelAdmin,
	"replSetFreeze":                  LevelAdmin,
	"replSetSyncFrom":                LevelAdmin,
	"replSetResizeOplog":             LevelAdmin,
	"addShard":                       LevelAdmin,
	"removeShard":                    LevelAdmin,
	"enableSharding":                 LevelAdmin,
	"shardCollection":                LevelAdmin,
	"unshardCollection":              LevelAdmin,
	"moveChunk":                      LevelAdmin,
	"splitChunk":                     LevelAdmin,
	"balancer":                       LevelAdmin,
	// eval 在服务端直接执行任意 JavaScript，能力等同管理员，不得按读命令放行
	"eval": LevelAdmin,
}

// scriptOperators 会让服务端执行任意 JavaScript 的操作符。
//
// 它们能把只读命令变成写入通道：`$where` 与 `$function`/`$accumulator` 的 JS 环境里能拿到 db 句柄，
// 从而改动其他集合。因此必须递归识别，不能只看命令名。
var scriptOperators = map[string]bool{
	"$where":       true,
	"$function":    true,
	"$accumulator": true,
}

// writeStages 聚合管道中会写出数据的 stage。aggregate 名义是读命令，含这些 stage 即为写。
var writeStages = map[string]bool{
	"$out":   true,
	"$merge": true,
}

// deleteOperators 写命令内部携带的删除动作：bulkWrite 的 deletes、findAndModify 的 remove。
// 命中即按「删文档」对待，否则 mongo:data:del 与 mongo:data:save 之间就出现了绕过面。
var deleteOperators = map[string]bool{
	"delete":  true,
	"deletes": true,
	"remove":  true,
}

// LevelOf 命令名的静态级别。未登记命令一律按 LevelAdmin（fail-closed）。
func LevelOf(name string) Level {
	if level, ok := commandLevels[name]; ok {
		return level
	}
	return LevelAdmin
}

// CommandName 命令文档的命令名，即第一个字段名。
func CommandName(command bson.D) (string, error) {
	if len(command) == 0 || command[0].Key == "" {
		return "", ErrEmptyCommand
	}
	return command[0].Key, nil
}

// Classify 判定命令文档的语义级别：静态表为基线，再按参数内容按需升级（取最大严重度）。
//
// explain 保持只读：服务端只做计划与统计，不会执行被解释的操作（包括写操作），
// 因此不因解释对象是写命令而抬级，否则只读账号无法用 explain 排查写性能。
func Classify(command bson.D) (string, Level, error) {
	name, err := CommandName(command)
	if err != nil {
		return "", LevelAdmin, err
	}

	level := LevelOf(name)

	// 只读命令的参数里含可执行 JS：具备写其他集合的能力，按写文档对待
	if level == LevelRead && hasKey(command, scriptOperators) {
		level = LevelDataSave
	}
	// 聚合含写出 stage：写文档
	if level == LevelRead && hasKey(command, writeStages) {
		level = LevelDataSave
	}
	// 写命令含删除动作：升为删文档
	if level == LevelDataSave && hasKey(command, deleteOperators) {
		level = LevelDataDel
	}
	return name, level, nil
}

// hasKey 递归查找键名命中集合 kvs 的字段。
//
// 容器类型必须穷举：bson.A（从 JSON 解出的数组）与 []bson.D（服务端自己组装的聚合管道）
// 都可能出现，漏一个就会让写出 stage 逃过分级判定（实测过：只扫 bson.A 时
// {$merge} 的聚合被当成只读放行）。
func hasKey(v any, kvs map[string]bool) bool {
	switch val := v.(type) {
	case bson.D:
		for _, e := range val {
			if kvs[e.Key] || hasKey(e.Value, kvs) {
				return true
			}
		}
	case bson.M:
		for key, item := range val {
			if kvs[key] || hasKey(item, kvs) {
				return true
			}
		}
	case bson.A:
		for _, item := range val {
			if hasKey(item, kvs) {
				return true
			}
		}
	case []bson.D:
		for _, item := range val {
			if hasKey(item, kvs) {
				return true
			}
		}
	case bson.E:
		return hasKey(val.Value, kvs)
	}
	return false
}

// CommandSpec 下发给前端的命令目录条目。
//
// 只给「事实与形状」：级别、要求的权限码、是否需要确认、模板与描述 i18n key。
// 纯皮肤（图标/颜色）由前端维护，与 redis 模块的职责边界一致。
type CommandSpec struct {
	Name        string `json:"name"`
	Level       string `json:"level"`
	Permission  string `json:"permission"`
	NeedConfirm bool   `json:"needConfirm"`
	Template    string `json:"template,omitempty"`
	DescKey     string `json:"descKey,omitempty"`
}

// commandTemplate 命令目录条目定义。
type commandTemplate struct {
	name     string
	template string
	descKey  string
}

// commandCatalog 控制台可选命令。收录范围刻意保守：只放排查类与常用管理动作；
// 未收录的命令依然可通过直接编辑命令 JSON 执行，并按 Classify 结果鉴权。
var commandCatalog = []commandTemplate{
	{"ping", `{"ping":1}`, "mongo.cmdPingDesc"},
	{"hello", `{"hello":1}`, "mongo.cmdHelloDesc"},
	{"serverStatus", `{"serverStatus":1}`, "mongo.cmdServerStatusDesc"},
	{"dbStats", `{"dbStats":1}`, "mongo.cmdDbStatsDesc"},
	{"collStats", `{"collStats":"<collection>"}`, "mongo.cmdCollStatsDesc"},
	{"listCollections", `{"listCollections":1}`, "mongo.cmdListCollectionsDesc"},
	{"listIndexes", `{"listIndexes":"<collection>"}`, "mongo.cmdListIndexesDesc"},
	{"indexStats", `{"indexStats":1,"indexStatOptions":{"all":true}}`, "mongo.cmdIndexStatsDesc"},
	{"validate", `{"validate":"<collection>"}`, "mongo.cmdValidateDesc"},
	{"currentOp", `{"currentOp":1}`, "mongo.cmdCurrentOpDesc"},
	{"usersInfo", `{"usersInfo":1}`, "mongo.cmdUsersInfoDesc"},
	{"rolesInfo", `{"rolesInfo":1,"showBuiltinRoles":true}`, "mongo.cmdRolesInfoDesc"},
	{"create", `{"create":"<collection>"}`, "mongo.cmdCreateDesc"},
	{"createIndexes", `{"createIndexes":"<collection>","indexes":[{"key":{"<field>":1},"name":"<indexName>"}]}`, "mongo.cmdCreateIndexesDesc"},
	{"collMod", `{"collMod":"<collection>","validator":{"$jsonSchema":{"bsonType":"object"}}}`, "mongo.cmdCollModDesc"},
	{"dropIndexes", `{"dropIndexes":"<collection>","index":"<indexName>"}`, "mongo.cmdDropIndexesDesc"},
	{"drop", `{"drop":"<collection>"}`, "mongo.cmdDropDesc"},
	{"dropDatabase", `{"dropDatabase":1}`, "mongo.cmdDropDatabaseDesc"},
	{"createUser", `{"createUser":"<username>","pwd":"<password>","roles":[{"role":"read","db":"<database>"}]}`, "mongo.cmdCreateUserDesc"},
	{"updateUser", `{"updateUser":"<username>","roles":[{"role":"read","db":"<database>"}]}`, "mongo.cmdUpdateUserDesc"},
	{"grantRolesToUser", `{"grantRolesToUser":"<username>","roles":[{"role":"<role>","db":"<database>"}]}`, "mongo.cmdGrantRolesToUserDesc"},
	{"dropUser", `{"dropUser":"<username>"}`, "mongo.cmdDropUserDesc"},
	{"setParameter", `{"setParameter":1,"<parameterName>":"<value>"}`, "mongo.cmdSetParameterDesc"},
}

// Catalog 命令目录（顺序即前端展示顺序）。
func Catalog() []*CommandSpec {
	specs := make([]*CommandSpec, 0, len(commandCatalog))
	for _, item := range commandCatalog {
		level := LevelOf(item.name)
		specs = append(specs, &CommandSpec{
			Name:        item.name,
			Level:       level.String(),
			Permission:  level.PermissionCode(),
			NeedConfirm: level.NeedConfirm(),
			Template:    item.template,
			DescKey:     item.descKey,
		})
	}
	return specs
}
