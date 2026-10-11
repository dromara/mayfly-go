//go:build it

package schema

// 结构对比集成测试的进程入口：负责临时库的统一清理。
//
// 与 dbm/itest、application 包同一套约定（各 IT 包自持连接工厂与收尾）：
// 由 scratchclean 为每个 (类型/宿主/端口/库名/账号/参数) 组合建一个随机命名的临时库，
// 进程结束时统一删库，测毕可用 information_schema 核验无残留。
//
// 运行：cd server && go test -tags it -count=1 -timeout 15m ./internal/db/application/schema/

import (
	"os"
	"testing"

	"mayfly-go/internal/db/ititest/scratchclean"

	_ "mayfly-go/internal/db/dbm/dialect/mysql"
	_ "mayfly-go/internal/db/dbm/dialect/postgres"
)

func TestMain(m *testing.M) {
	os.Exit(scratchclean.Run(m.Run))
}
