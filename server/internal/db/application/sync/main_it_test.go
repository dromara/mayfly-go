//go:build it

package sync

import (
	"context"
	"os"
	"testing"

	"mayfly-go/internal/db/ititest/scratchclean"
	"mayfly-go/pkg/rediscli"

	"github.com/redis/go-redis/v9"
)

// TestMain 使用进程独占临时库，测试结束无条件清理，清理失败使测试失败。
// 顺带初始化 Redis（127.0.0.1:6332），跨实例停止标记用例依赖之；Redis 不可达时保持全局 cli 为 nil，
// 由具体用例 t.Skip，不影响其他只依赖 mysql/pg 的用例。
func TestMain(m *testing.M) {
	cli := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6332"})
	if _, err := cli.Ping(context.Background()).Result(); err == nil {
		rediscli.SetCli(cli)
	}
	os.Exit(scratchclean.Run(m.Run))
}
