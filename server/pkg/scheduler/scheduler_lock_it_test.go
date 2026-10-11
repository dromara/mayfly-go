package scheduler

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/pkg/rediscli"
)

// 与 pkg/rediscli、pkg/taskx 的用例同一约定：本机无 Redis 时整包跳过而不是失败，
// 但有 Redis 时跨实例语义必须真跑（这条链路一旦被跳过就永远没人验证它）
func TestMain(m *testing.M) {
	cli := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6332"})
	if _, err := cli.Ping(context.Background()).Result(); err != nil {
		fmt.Println("SKIP: Redis not available at 127.0.0.1:6332")
		os.Exit(0)
	}
	rediscli.SetCli(cli)
	code := m.Run()
	for _, pattern := range []string{"mayfly:lock:scheduler:ittest-*"} {
		if keys, _ := cli.Keys(context.Background(), pattern).Result(); len(keys) > 0 {
			cli.Del(context.Background(), keys...)
		}
	}
	cli.Close()
	os.Exit(code)
}

// TestAddFunByKeyWithLockSkipsWhenOtherInstanceHoldsLock 别的实例正在跑这一轮时，本实例一次都不能跑。
//
// 审批回放的清扫任务、告警评估等 9 个定时任务全靠这条入口做跨实例互斥：这里若不去判锁，
// 两个实例会把同一批任务各跑一遍（重复下发命令、重复评估告警）。用「先占住 scheduler:<key> 的锁」
// 模拟另一个实例，断言任务体一次都没执行；对方释放后必须恢复执行，否则就成了「永远没人跑」
func TestAddFunByKeyWithLockSkipsWhenOtherInstanceHoldsLock(t *testing.T) {
	const key = "ittest-lock-held"

	other := rediscli.NewLock("scheduler:"+key, 3*time.Second)
	require.NotNil(t, other)
	require.True(t, other.Lock(), "测试前提：模拟的实例要先占到锁")

	var runs atomic.Int64
	require.NoError(t, AddFunByKeyWithLock(key, "@every 100ms", time.Minute, func() { runs.Add(1) }))
	defer RemoveByKey(key)

	time.Sleep(450 * time.Millisecond)
	assert.Zero(t, runs.Load(), "锁被其他实例持有时本实例不得执行任务体")

	other.UnLock()

	deadline := time.Now().Add(5 * time.Second)
	for runs.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	assert.Greater(t, runs.Load(), int64(0), "对方释放锁后本实例必须接着跑，不能永远停在跳过")
}

// TestAddFunByKeyWithLockReleasesAfterRun 每轮跑完必须立刻释放锁，否则锁的 TTL 会变成「两次执行之间的强制间隔」
func TestAddFunByKeyWithLockReleasesAfterRun(t *testing.T) {
	const key = "ittest-lock-release"

	ran := make(chan struct{}, 4)
	// TTL 远大于执行间隔：若只在 TTL 到期后才放开，这里第二轮就跑不起来
	require.NoError(t, AddFunByKeyWithLock(key, "@every 120ms", time.Hour, func() {
		ran <- struct{}{}
	}))
	defer RemoveByKey(key)

	for round := 0; round < 2; round++ {
		select {
		case <-ran:
		case <-time.After(3 * time.Second):
			t.Fatalf("第 %d 轮没有执行：任务体跑完后锁没有被释放，长 TTL 把后续轮次全部挡掉了", round+1)
		}
	}
}
