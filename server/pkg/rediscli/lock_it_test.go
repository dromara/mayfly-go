package rediscli

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 与 pkg/taskx 的 RunGuard 用例同一约定：本机 Redis 不可达时整包跳过而不是失败，
// 因为 CI 上可能没有 Redis；但一旦有，跨实例语义必须真跑
func TestMain(m *testing.M) {
	cli := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6332"})
	if _, err := cli.Ping(context.Background()).Result(); err != nil {
		fmt.Println("SKIP: Redis not available at 127.0.0.1:6332")
		os.Exit(0)
	}
	SetCli(cli)
	code := m.Run()
	for _, pattern := range []string{"mayfly:lock:ittest-*"} {
		if keys, _ := cli.Keys(context.Background(), pattern).Result(); len(keys) > 0 {
			cli.Del(context.Background(), keys...)
		}
	}
	cli.Close()
	os.Exit(code)
}

// TestRedisLockIsOwnerCheckedOnUnlock 锁被接管后，原持有者的 UnLock 不得删掉新持有者的锁。
//
// 这是定时任务跨实例互斥唯一的失效点：任务跑得比锁的 TTL 久时锁会自然过期、被另一个实例拿走，
// 此时本实例收尾若无条件 DEL，锁就等于不存在——两个实例从此同时跑同一批任务（清扫/评估会重复下发）。
// 现象不在本次运行里暴露，而是等到下一次并发时才出现，所以必须有用例钉住
func TestRedisLockIsOwnerCheckedOnUnlock(t *testing.T) {
	const key = "ittest-lock-takeover"

	first := NewLock(key, 120*time.Millisecond)
	require.NotNil(t, first)
	require.True(t, first.Lock(), "空闲锁必须能拿到")

	time.Sleep(250 * time.Millisecond)

	second := NewLock(key, time.Minute)
	require.True(t, second.Lock(), "锁过期后必须能被其他实例接管")

	assert.False(t, first.UnLock(), "原持有者不得删掉别人刚拿到的锁")
	assert.False(t, second.Lock(), "锁仍在第二方手里，不能被再次抢到")

	exists, err := GetCli().Exists(context.Background(), LockKeyPrefix+key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "被接管的锁必须还在")

	assert.True(t, second.UnLock(), "真正的持有者应当能释放")
}

// TestRedisLockTtlIsApplied 锁必须带存活时间，且活着时抢不到、释放/过期后可被接管。
//
// 进程崩溃后残留的锁若永不过期，该定时任务在所有实例上都会被永久跳过
// （表现为「功能再也没有自动跑过」，界面上毫无线索）
func TestRedisLockTtlIsApplied(t *testing.T) {
	const key = "ittest-lock-ttl"

	lock := NewLock(key, 2*time.Second)
	require.NotNil(t, lock)
	require.True(t, lock.Lock())

	// Redis 的 TTL 命令只回报整秒，亚秒级锁会被读成 0（实测），因此这里用 PTTL 看毫秒
	ttl, err := GetCli().PTTL(context.Background(), LockKeyPrefix+key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl.Milliseconds(), int64(0), "锁必须有存活时间，不能永久占位")
	assert.LessOrEqual(t, ttl.Milliseconds(), int64(2000), "存活时间必须等于给定的 TTL")

	assert.False(t, NewLock(key, time.Minute).Lock(), "锁还活着时其他实例不得抢到")

	require.True(t, lock.UnLock(), "持有者本人应能释放")
	takeover := NewLock(key, time.Second)
	assert.True(t, takeover.Lock(), "释放后必须能被其他实例接管")
	assert.False(t, NewLock(key, time.Second).Lock(), "接管者持有期间不得被再次抢走")
}
