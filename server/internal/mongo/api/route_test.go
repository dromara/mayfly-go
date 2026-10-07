package api

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"mayfly-go/internal/mongo/application/mongodoc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

// TestRouteRegistration 路由注册冒烟测试。
//
// gin 在同一路由树上遇到参数名不一致等冲突会直接 panic，这里让冲突在测试期暴露而不是等服务启动；
// 同时把「数据面接口必须全部注册成功」钉住：缺一个就意味着前端对应操作会 404。
func TestRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	confs := (&Mongo{}).ReqConfs()
	req.BatchSetGroup(engine.Group("/api"+confs.Group), confs.Confs)

	routes := make([]string, 0, 32)
	for _, route := range engine.Routes() {
		routes = append(routes, route.Method+" "+route.Path)
	}
	sort.Strings(routes)

	for _, want := range []string{
		"DELETE /api/mongos/:id",
		"DELETE /api/mongos/:id/databases/:db",
		"DELETE /api/mongos/:id/databases/:db/collections/:coll",
		"GET /api/mongos",
		"GET /api/mongos/:id/commands",
		"GET /api/mongos/:id/databases",
		"GET /api/mongos/:id/databases/:db/collections",
		"GET /api/mongos/:id/databases/:db/collections/:coll/meta",
		"POST /api/mongos",
		"POST /api/mongos/:id/aggregate",
		"POST /api/mongos/:id/databases/:db/collections",
		"POST /api/mongos/:id/databases/:db/collections/:coll/indexes",
		"DELETE /api/mongos/:id/databases/:db/collections/:coll/indexes/:index",
		"POST /api/mongos/:id/docs",
		"POST /api/mongos/:id/docs/delete",
		"POST /api/mongos/:id/docs/delete-by-filter",
		"POST /api/mongos/:id/docs/update",
		"POST /api/mongos/:id/export",
		"POST /api/mongos/:id/query",
		"POST /api/mongos/:id/run-command",
		"POST /api/mongos/test-conn",
		"PUT /api/mongos/:id/doc",
	} {
		if !containsStr(routes, want) {
			t.Fatalf("route [%s] is not registered, routes: %s", want, strings.Join(routes, ", "))
		}
	}

	// 被取代的旧数据面入口不允许复活：它们以 map[string]any 承载文档，
	// 既丢字段顺序（复合排序键随机化）又丢 BSON 类型（读写往返静默损坏）
	for _, gone := range []string{
		"POST /api/mongos/:id/command/find",
		"POST /api/mongos/:id/command/insert",
		"POST /api/mongos/:id/command/update-by-id",
		"POST /api/mongos/:id/command/delete-by-id",
		"GET /api/mongos/:id/collections",
	} {
		if containsStr(routes, gone) {
			t.Fatalf("route [%s] should be removed, it is replaced by the typed data-plane api", gone)
		}
	}
}

// nonStaticPermRoutes 不声明静态权限码的 POST/PUT/DELETE 路由及其理由。
//
// 路由方法不等于操作语义：这里逐项登记的都是「形式上是 POST/PUT/DELETE、语义上不是写」或
// 「鉴权口径无法静态确定」的入口。新增此类路由必须同时在此登记理由，否则本测试失败；
// 豁免为动态鉴权的入口还必须真的在 handler 内做分级校验（见末尾断言）。
var nonStaticPermRoutes = map[string]string{
	`"/test-conn"`: "只探测连通性，不改动任何数据",
	// 查询条件文档可能很长且含任意符号，只能走请求体而不能走 query string
	`":id/query"`: "只读查询，仅用 POST 承载 filter/sort/projection 文档",
	// 聚合是否写数据取决于 stage：含 $out/$merge 才是写操作，explain 根本不执行
	`":id/aggregate"`: "聚合权限取决于 stage（写出/只读），由 handler 内 requireCmdPerm 动态判定",
	// 透传型入口能做什么都取决于命令本身，权限必须按命令语义动态判定
	`":id/run-command"`: "透传型入口，权限取决于命令语义，由 handler 内 requireCmdPerm 动态判定",
}

// TestMutatingRoutesDeclarePermission 每个会改数据的静态路由都必须声明权限码。
//
// 这是权限绕过的防回潮门禁：新增写接口时忘记 RequiredPermissionCode 只会在生产上表现为
// 「任何看得到该资源的人都能写」，运行时不会报错，因此在测试期按路由表源码逐条核对。
func TestMutatingRoutesDeclarePermission(t *testing.T) {
	source := readApiSource(t, "mongo.go")

	var exempted []string
	mutating := regexp.MustCompile(`req\.New(?:Post|Put|Delete)\(\s*("[^"]*")`)
	declared := regexp.MustCompile(`RequiredPermissionCode\(`)

	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		routeMatch := mutating.FindStringSubmatch(line)
		if routeMatch == nil || declared.MatchString(line) {
			continue
		}
		if reason, ok := nonStaticPermRoutes[routeMatch[1]]; ok {
			exempted = append(exempted, routeMatch[1])
			t.Logf("exempted %s: %s", routeMatch[1], reason)
			continue
		}
		t.Errorf("mutating route %s must declare a permission code", routeMatch[1])
	}

	// 防空跑：白名单里的每一项都必须在路由表里真实出现，避免豁免项随重构演变成死条目
	for path := range nonStaticPermRoutes {
		if !containsStr(exempted, path) {
			t.Errorf("permission exemption %s matched nothing, the route table changed without updating this test", path)
		}
	}

	if content := readApiSource(t, "mongo_cmd.go"); !strings.Contains(content, "m.requireCmdPerm(") {
		t.Error("run-command must classify the command and enforce the matching permission in-handler")
	}
	// 聚合也走同一套动态判定，不能只靠「静态一个码」糊弄过去
	if content := readApiSource(t, "mongo_ops.go"); !strings.Contains(content, "m.requireCmdPerm(") {
		t.Error("aggregate must classify its pipeline and enforce the matching permission in-handler")
	}
	for _, file := range []string{"mongo_cmd.go", "mongo_ops.go"} {
		if content := readApiSource(t, file); strings.Contains(content, "requireCmdPerm(rc, \"aggregate\", mongodoc.LevelRead") {
			t.Errorf("%s must not hardcode the aggregate permission level, derive it from Classify", file)
		}
	}
}

// TestPermissionCodeLiterals 权限码字面值属于跨语言契约的一部分。
//
// 角色管理里已勾选的资源行（t_sys_resource.code）与前端 v-auth 都按字符串匹配，
// 后端改常量值不会有任何报错，只会让既有授权静默失效，因此把字面值钉在测试里。
func TestPermissionCodeLiterals(t *testing.T) {
	cases := map[string]string{
		mongodoc.PermDataSave: "mongo:data:save",
		mongodoc.PermDataDel:  "mongo:data:del",
		mongodoc.PermDDLSave:  "mongo:ddl:save",
		mongodoc.PermDDLDel:   "mongo:ddl:del",
		mongodoc.PermCmdAdmin: "mongo:cmd:admin",
		PermDataExport:        "mongo:data:export",
	}
	for got, want := range cases {
		if got != want {
			t.Fatalf("permission code literal changed: got %q, want %q", got, want)
		}
	}

	// 只读命令不要求权限码：它只受资源可见性约束，多一条权限反而让「能看不能查」这种状态无法表达
	if code := mongodoc.LevelRead.PermissionCode(); code != "" {
		t.Fatalf("read level must not require a permission code, got %q", code)
	}
}

func readApiSource(t *testing.T, file string) string {
	t.Helper()

	// 只允许读取本包内固定的几个 .go 源文件，避免任何带路径分隔符的名字进入文件读取
	if !strings.HasSuffix(file, ".go") || strings.ContainsAny(file, "/\\") {
		t.Fatalf("unexpected source file name: %q", file)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	return string(data)
}

func containsStr(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
