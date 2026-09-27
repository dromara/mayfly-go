package api

import (
	"mayfly-go/pkg/req"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRouteRegistration 路由注册冒烟测试：
// gin 在同一路由树上遇到参数名不一致等冲突会直接 panic，这里让冲突在测试期暴露，而不是等服务启动
func TestRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	confs := (&Redis{}).ReqConfs()
	req.BatchSetGroup(engine.Group("/api"+confs.Group), confs.Confs)

	routes := make([]string, 0, 32)
	for _, route := range engine.Routes() {
		routes = append(routes, route.Method+" "+route.Path)
	}
	sort.Strings(routes)

	// 类型化数据面接口必须全部注册成功，缺一个就意味着前端对应操作会 404
	for _, want := range []string{
		"GET /api/redis/:id/:db/views",
		"GET /api/redis/:id/:db/commands",
		"GET /api/redis/:id/:db/key-meta",
		"POST /api/redis/:id/:db/key-values",
		"PUT /api/redis/:id/:db/key-value",
		"POST /api/redis/:id/:db/key-op",
		"POST /api/redis/:id/:db/key-summary",
		"PUT /api/redis/:id/:db/key-ttl",
		"PUT /api/redis/:id/:db/key-rename",
		"POST /api/redis/:id/:db/key-copy",
		"POST /api/redis/:id/:db/del-keys",
	} {
		if !contains(routes, want) {
			t.Fatalf("route [%s] is not registered, routes: %s", want, strings.Join(routes, ", "))
		}
	}

	// 被 key-meta 取代的单值接口不允许复活（同一份数据只允许有一个读取入口）
	for _, gone := range []string{
		"GET /api/redis/:id/:db/key-info",
		"GET /api/redis/:id/:db/key-ttl",
		"GET /api/redis/:id/:db/key-memuse",
	} {
		if contains(routes, gone) {
			t.Fatalf("route [%s] should be removed, it duplicates key-meta", gone)
		}
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
