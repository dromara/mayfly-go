package mongodoc

import (
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestClassify 透传命令入口的鉴权判据。
//
// 这里锁死两类真实绕过面：一是命令名之外的「可执行内容」（$where / $function / $out / $merge），
// 二是写命令里夹带的删除动作（bulkWrite.deletes / findAndModify.remove）。
// 只看命令名做分级等于把「只读账号删库」留成缺口。
func TestClassify(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    Level
	}{
		{"查询是只读", `{"find":"orders"}`, LevelRead},
		{"聚合默认只读", `{"aggregate":"orders","pipeline":[{"$match":{"a":1}}]}`, LevelRead},
		{"聚合含 $out 即写", `{"aggregate":"orders","pipeline":[{"$match":{"a":1}},{"$out":"backup"}]}`, LevelDataSave},
		{"聚合含 $merge 即写", `{"aggregate":"orders","pipeline":[{"$merge":{"into":"t2"}}]}`, LevelDataSave},
		{"find 含 $where 可执行 JS，按写处理", `{"find":"orders","filter":{"$where":"db.users.insert({})"}}`, LevelDataSave},
		{"聚合含 $function 可执行 JS，按写处理", `{"aggregate":"orders","pipeline":[{"$project":{"v":{"$function":{"body":"() => 1"}}}}]}`, LevelDataSave},
		{"explain 不执行操作，保持只读", `{"explain":{"delete":"orders","deletes":[{"q":{}}]},"verbosity":"plan"}`, LevelRead},
		{"插入是写", `{"insert":"orders","documents":[{}]}`, LevelDataSave},
		{"删除命令是删文档", `{"delete":"orders","deletes":[{"q":{"a":1}}]}`, LevelDataDel},
		{"bulkWrite 含 deletes 按删除处理", `{"bulkWrite":"orders","ops":[{"delete":{"q":{}}}]}`, LevelDataDel},
		{"bulkWrite 只有插入按写处理", `{"bulkWrite":"orders","ops":[{"insert":{"document":{"a":1}}}]}`, LevelDataSave},
		{"findAndModify remove 按删除处理", `{"findAndModify":"orders","remove":true}`, LevelDataDel},
		{"findAndModify update 按写处理", `{"findAndModify":"orders","update":{"$set":{"a":1}}}`, LevelDataSave},
		{"建集合是结构变更", `{"create":"orders"}`, LevelStructSave},
		{"建索引是结构变更", `{"createIndexes":"orders","indexes":[{"key":{"a":1}}]}`, LevelStructSave},
		{"删索引销毁结构", `{"dropIndexes":"orders","index":"a_1"}`, LevelStructDel},
		{"删集合销毁结构", `{"drop":"orders"}`, LevelStructDel},
		{"删库销毁结构", `{"dropDatabase":1}`, LevelStructDel},
		{"建账号是管理", `{"createUser":"u1","pwd":"p"}`, LevelAdmin},
		{"grantRolesToUser 是管理", `{"grantRolesToUser":"u1","roles":["readWrite"]}`, LevelAdmin},
		{"eval 服务端任意 JS，管理级", `{"eval":"db.orders.drop()"}`, LevelAdmin},
		{"未知命令 fail-closed 到管理级", `{"someFutureCommand":1}`, LevelAdmin},
		{"大小写异常写法不做归一，落到未知分支", `{"DROP":"orders"}`, LevelAdmin},
		{"带引号差异的命令名同样落到未知分支", `{"drop ":"orders"}`, LevelAdmin},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc, err := Decode(json.RawMessage(c.command))
			if err != nil {
				t.Fatalf("decode command %q: %v", c.command, err)
			}
			_, level, err := Classify(doc)
			if err != nil {
				t.Fatalf("classify %q: %v", c.command, err)
			}
			if level != c.want {
				t.Fatalf("classify(%s) = %s, want %s", c.command, level, c.want)
			}
		})
	}
}

func TestClassifyEmptyCommand(t *testing.T) {
	if _, _, err := Classify(bson.D{}); err == nil {
		t.Fatal("empty command must be rejected")
	}
}

// TestLevelPermissionMapping 每个非只读级别都必须映射到一个独立且非空的权限码，
// 否则「分级」在鉴权层面等于没有分级。
func TestLevelPermissionMapping(t *testing.T) {
	levels := []Level{LevelRead, LevelDataSave, LevelDataDel, LevelStructSave, LevelStructDel, LevelAdmin}

	seen := map[string]Level{}
	for _, level := range levels {
		code := level.PermissionCode()
		if level == LevelRead {
			if code != "" {
				t.Fatalf("read level must not require a permission code, got %q", code)
			}
			continue
		}
		if code == "" {
			t.Fatalf("level %s must require a permission code", level)
		}
		if dup, ok := seen[code]; ok {
			t.Fatalf("levels %s and %s share permission code %q", dup, level, code)
		}
		seen[code] = level
	}

	if len(seen) != len(levels)-1 {
		t.Fatalf("permission codes not 1:1 with levels: %v", seen)
	}
}

// TestLevelNeedConfirm 只读与常规写入不需要确认；会销毁数据或越出当前集合范围的一律需要。
func TestLevelNeedConfirm(t *testing.T) {
	cases := map[Level]bool{
		LevelRead:       false,
		LevelDataSave:   false,
		LevelDataDel:    true,
		LevelStructSave: true,
		LevelStructDel:  true,
		LevelAdmin:      true,
	}
	for level, want := range cases {
		if got := level.NeedConfirm(); got != want {
			t.Fatalf("NeedConfirm(%s) = %v, want %v", level, got, want)
		}
	}
}

func TestLevelString(t *testing.T) {
	cases := map[Level]string{
		LevelRead:       "read",
		LevelDataSave:   "dataSave",
		LevelDataDel:    "dataDel",
		LevelStructSave: "structSave",
		LevelStructDel:  "structDel",
		LevelAdmin:      "admin",
		Level(99):       "admin", // 越界值也必须报出最高级，不能落到空串
	}
	for level, want := range cases {
		if got := level.String(); got != want {
			t.Fatalf("Level(%d).String() = %s, want %s", level, got, want)
		}
	}
}

// TestCatalogTemplates 命令目录的每条模板必须满足：
//  1. 是合法的 JSON 命令文档；
//  2. 第一个字段名与条目声明的命令名一致（写错命令名会让前端下拉项执行到另一条命令）；
//  3. 下发的级别、权限码与确认标记与分级表同源，不在前端二次推导。
//
// 这是目录与分级表之间的跨文件契约测试，新增命令漏登记会在此被拦下。
func TestCatalogTemplates(t *testing.T) {
	specs := Catalog()
	if len(specs) == 0 {
		t.Fatal("command catalog is empty")
	}

	for _, spec := range specs {
		doc, err := Decode(json.RawMessage(spec.Template))
		if err != nil {
			t.Fatalf("command %q template is invalid: %v (%s)", spec.Name, err, spec.Template)
		}
		name, level, err := Classify(doc)
		if err != nil {
			t.Fatalf("command %q classify: %v", spec.Name, err)
		}
		if name != spec.Name {
			t.Fatalf("command %q template declares wrong name %q: %s", spec.Name, name, spec.Template)
		}
		if level != LevelOf(spec.Name) {
			t.Fatalf("command %q level drift: template says %s, table says %s", spec.Name, level, LevelOf(spec.Name))
		}
		if spec.Level != level.String() {
			t.Fatalf("command %q spec level %s != classified %s", spec.Name, spec.Level, level)
		}
		if spec.Permission != level.PermissionCode() {
			t.Fatalf("command %q permission %s != %q", spec.Name, spec.Permission, level.PermissionCode())
		}
		if spec.NeedConfirm != level.NeedConfirm() {
			t.Fatalf("command %q needConfirm = %v, want %v", spec.Name, spec.NeedConfirm, level.NeedConfirm())
		}
		if spec.DescKey == "" {
			t.Fatalf("command %q has no i18n description key", spec.Name)
		}
	}
}

// TestCatalogCoversDangerousCommands 目录里必须给出破坏性命令的说明条目：
// 前端拿不到目录条目时只能盲输命令，那正是「找不到即等于功能丢失」的反面。
func TestCatalogCoversDangerousCommands(t *testing.T) {
	index := map[string]*CommandSpec{}
	for _, spec := range Catalog() {
		index[spec.Name] = spec
	}

	for _, name := range []string{"drop", "dropDatabase", "dropIndexes", "createUser", "dropUser", "grantRolesToUser"} {
		spec, ok := index[name]
		if !ok {
			t.Fatalf("command %q missing from catalog", name)
		}
		if !spec.NeedConfirm {
			t.Fatalf("command %q must require confirmation", name)
		}
		if spec.Permission == "" {
			t.Fatalf("command %q must declare a permission code", name)
		}
	}
}

// TestCommandTableNamesNotCatalogOnly 分级表与目录的一致性：
// 目录是分级表的子集，反向不成立（未收录命令仍可手输执行并鉴权）。
func TestCommandTableNamesNotCatalogOnly(t *testing.T) {
	cataloged := map[string]bool{}
	for _, item := range commandCatalog {
		if _, ok := commandLevels[item.name]; !ok {
			t.Fatalf("catalog command %q is not registered in the level table", item.name)
		}
		cataloged[item.name] = true
	}
	if len(cataloged) != len(commandCatalog) {
		t.Fatal("duplicated command name in catalog")
	}
}
