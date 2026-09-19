package taskx

import (
	"context"
	"fmt"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/scheduler"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	cli := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6332",
	})
	if _, err := cli.Ping(ctx).Result(); err != nil {
		fmt.Println("SKIP: Redis not available at 127.0.0.1:6332")
		os.Exit(0)
	}
	rediscli.SetCli(cli)
	code := m.Run()
	// 清理测试 key
	keys, _ := cli.Keys(ctx, "mayfly:run-guard:ittest*").Result()
	if len(keys) > 0 {
		cli.Del(ctx, keys...)
	}
	cli.Close()
	os.Exit(code)
}

// ========== CasDel Tests ==========

func TestCasDel_MatchingValue(t *testing.T) {
	key := "mayfly:lock:ittest-casdel-match"
	defer rediscli.GetCli().Del(ctx, key)

	require.NoError(t, rediscli.GetCli().Set(ctx, key, "abc123", 0).Err())
	assert.True(t, rediscli.CasDel(key, "abc123"), "值匹配应删除成功")

	exists, _ := rediscli.GetCli().Exists(ctx, key).Result()
	assert.Equal(t, int64(0), exists, "删除后 key 不应存在")
}

func TestCasDel_NonMatchingValue(t *testing.T) {
	key := "mayfly:lock:ittest-casdel-nomatch"
	defer rediscli.GetCli().Del(ctx, key)

	require.NoError(t, rediscli.GetCli().Set(ctx, key, "abc123", 0).Err())
	assert.False(t, rediscli.CasDel(key, "xyz789"), "值不匹配应返回 false")

	exists, _ := rediscli.GetCli().Exists(ctx, key).Result()
	assert.Equal(t, int64(1), exists, "值不匹配时 key 应保留")
}

func TestCasDel_NonExistentKey(t *testing.T) {
	key := "mayfly:lock:ittest-casdel-noexist"
	defer rediscli.GetCli().Del(ctx, key)

	assert.False(t, rediscli.CasDel(key, "any"), "key 不存在应返回 false")
}

// ========== RunGuard + Redis: 基本生命周期 ==========

func TestRunGuard_Redis_AcquireReleaseCycle(t *testing.T) {
	g := &RunGuard[uint64]{}
	key := "mayfly:run-guard:42"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	assert.True(t, g.Acquire(42), "首次 Acquire 应成功")

	// 验证 Redis key 存在
	exists, _ := rediscli.GetCli().Exists(ctx, key).Result()
	assert.Equal(t, int64(1), exists, "Acquire 后 Redis key 应存在")

	// 验证 IsRunning
	assert.True(t, g.IsRunning(42))

	// Release
	g.Release(42)
	assert.False(t, g.IsRunning(42))

	// 验证 Redis key 已清理
	exists, _ = rediscli.GetCli().Exists(ctx, key).Result()
	assert.Equal(t, int64(0), exists, "Release 后 Redis key 应已删除")
}

func TestRunGuard_Redis_ReacquireAfterRelease(t *testing.T) {
	g := &RunGuard[string]{}
	defer rediscli.GetCli().Del(ctx, "mayfly:run-guard:task-x")

	assert.True(t, g.Acquire("task-x"))
	g.Release("task-x")
	assert.True(t, g.Acquire("task-x"), "释放后应可重新获取")
	g.Release("task-x")
}

// ========== RunGuard + Redis: 跨实例互斥 ==========

func TestRunGuard_Redis_CrossInstanceMutex(t *testing.T) {
	// 模拟两个独立实例（两个 RunGuard 共享同一 Redis）
	g1 := &RunGuard[uint64]{}
	g2 := &RunGuard[uint64]{}
	key := "mayfly:run-guard:100"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	assert.True(t, g1.Acquire(100), "实例1 应获取成功")
	assert.False(t, g2.Acquire(100), "实例2 应获取失败（跨实例互斥）")

	// g2 的 IsRunning 应通过 Redis 检测到 g1 的运行态
	assert.True(t, g2.IsRunning(100), "实例2 应通过 Redis 感知到实例1 正在运行")

	g1.Release(100)
	assert.True(t, g2.Acquire(100), "实例1 释放后实例2 应获取成功")
	g2.Release(100)
}

func TestRunGuard_Redis_ConcurrentCrossInstance(t *testing.T) {
	// 64 个 goroutine 分属不同 RunGuard（模拟多实例），仅一个应成功
	const workers = 64
	guards := make([]*RunGuard[uint64], workers)
	for i := range guards {
		guards[i] = &RunGuard[uint64]{}
	}
	key := "mayfly:run-guard:999"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	var acquired int32
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if guards[idx].Acquire(999) {
				mu.Lock()
				acquired++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int32(1), acquired, "跨实例并发 Acquire 仅应一个成功")
}

// ========== RunGuard + Redis: 所有权验证 ==========

func TestRunGuard_Redis_OwnershipProtection(t *testing.T) {
	// 场景：g1 的锁 TTL 过期后被 g2 获取，g1 的 Release 不应删除 g2 的锁
	g1 := &RunGuard[string]{LockTTL: 2 * time.Second}
	g2 := &RunGuard[string]{LockTTL: 10 * time.Second}
	key := "mayfly:run-guard:owner-test"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	// g1 获取锁（短 TTL）
	assert.True(t, g1.Acquire("owner-test"))

	// 等待 TTL 过期
	time.Sleep(2500 * time.Millisecond)

	// g2 获取同一把锁（g1 的 Redis key 已过期）
	assert.True(t, g2.Acquire("owner-test"), "g1 TTL 过期后 g2 应获取成功")

	// g1 Release：CasDel 值不匹配，不应删除 g2 的锁
	g1.Release("owner-test")

	// 验证 g2 的锁仍在
	exists, _ := rediscli.GetCli().Exists(ctx, key).Result()
	assert.Equal(t, int64(1), exists, "g1 的 Release 不应误删 g2 的锁")

	// g2 仍可正常释放
	g2.Release("owner-test")
	exists, _ = rediscli.GetCli().Exists(ctx, key).Result()
	assert.Equal(t, int64(0), exists, "g2 Release 后 key 应删除")
}

// ========== RunGuard + Redis: TTL 过期 ==========

func TestRunGuard_Redis_TTLExpiry(t *testing.T) {
	g := &RunGuard[uint64]{LockTTL: 2 * time.Second}
	key := "mayfly:run-guard:777"
	rediscli.GetCli().Del(ctx, key) // 确保干净状态
	defer rediscli.GetCli().Del(ctx, key)

	require.True(t, g.Acquire(777), "Acquire 应成功")

	// 验证 TTL
	ttl, _ := rediscli.GetCli().TTL(ctx, key).Result()
	assert.True(t, ttl > 0 && ttl <= 2*time.Second, "TTL 应约为 2s，实际: %v", ttl)

	// 等待 TTL 过期
	time.Sleep(2500 * time.Millisecond)

	// Redis key 应已过期
	exists, _ := rediscli.GetCli().Exists(ctx, key).Result()
	assert.Equal(t, int64(0), exists, "TTL 过期后 Redis key 应不存在")

	// 本地 held 仍有记录（未 Release），IsRunning 通过 Redis 检查应返回 false
	// 但本地 map 仍有记录 → IsRunning 返回 true（本地优先）
	assert.True(t, g.IsRunning(777), "本地 held 未清理，IsRunning 应返回 true")

	// Release 清理本地 + CasDel 不匹配（key 已过期）→ 安全
	g.Release(777)
}

// ========== RunGuard + Redis: 默认 TTL ==========

func TestRunGuard_Redis_DefaultTTL(t *testing.T) {
	g := &RunGuard[uint64]{} // LockTTL 为零 → 使用默认 2h
	key := "mayfly:run-guard:888"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	assert.True(t, g.Acquire(888))

	ttl, _ := rediscli.GetCli().TTL(ctx, key).Result()
	assert.True(t, ttl > 119*time.Minute && ttl <= 2*time.Hour,
		"默认 TTL 应约为 2h，实际: %v", ttl)

	g.Release(888)
}

// ========== RunGuard + Redis: 亚秒级 TTL 防护 ==========

func TestRunGuard_Redis_SubSecondTTLProtection(t *testing.T) {
	// go-redis 将 sub-second duration 截断为 0 秒（Redis 中 0 = 永不过期）
	// lockTTL() 应拒绝亚秒级值，回退到默认 2h
	g := &RunGuard[uint64]{LockTTL: 500 * time.Millisecond}
	key := "mayfly:run-guard:666"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	require.True(t, g.Acquire(666))

	ttl, _ := rediscli.GetCli().TTL(ctx, key).Result()
	// 应回退到默认 2h，而非 0（永不过期）
	assert.True(t, ttl > 119*time.Minute, "亚秒级 TTL 应回退到默认 2h，实际: %v", ttl)

	g.Release(666)
}

// ========== RunGuard + Redis: Release 幂等 ==========

func TestRunGuard_Redis_ReleaseIdempotent(t *testing.T) {
	g := &RunGuard[uint64]{}
	key := "mayfly:run-guard:555"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	assert.True(t, g.Acquire(555))

	// 多次 Release 不应 panic 或报错
	g.Release(555)
	g.Release(555)
	g.Release(555)

	exists, _ := rediscli.GetCli().Exists(ctx, key).Result()
	assert.Equal(t, int64(0), exists, "多次 Release 后 key 应不存在")
}

// ========== RunGuard + Redis: 不同任务隔离 ==========

func TestRunGuard_Redis_TaskIsolation(t *testing.T) {
	g := &RunGuard[string]{}
	defer func() {
		rediscli.GetCli().Del(ctx, "mayfly:run-guard:iso-A", "mayfly:run-guard:iso-B")
	}()

	assert.True(t, g.Acquire("iso-A"))
	assert.True(t, g.Acquire("iso-B"))

	// 两个 Redis key 应独立存在
	existsA, _ := rediscli.GetCli().Exists(ctx, "mayfly:run-guard:iso-A").Result()
	existsB, _ := rediscli.GetCli().Exists(ctx, "mayfly:run-guard:iso-B").Result()
	assert.Equal(t, int64(1), existsA)
	assert.Equal(t, int64(1), existsB)

	// 释放 A 不影响 B
	g.Release("iso-A")
	existsA, _ = rediscli.GetCli().Exists(ctx, "mayfly:run-guard:iso-A").Result()
	existsB, _ = rediscli.GetCli().Exists(ctx, "mayfly:run-guard:iso-B").Result()
	assert.Equal(t, int64(0), existsA)
	assert.Equal(t, int64(1), existsB)

	g.Release("iso-B")
}

// ========== RunGuard + Redis: Redis 故障/降级场景 ==========

func TestRunGuard_Redis_LocalOnlyWhenRedisUnavailable(t *testing.T) {
	// Redis 不可用时走本地降级路径：Acquire 成功、IsRunning 仅检查本地
	g := &RunGuard[uint64]{}
	key := "mayfly:run-guard:12345"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	cli := rediscli.GetCli()
	rediscli.SetCli(nil)
	defer rediscli.SetCli(cli)

	// 本地降级路径：Acquire 应成功
	assert.True(t, g.Acquire(12345), "Redis 不可用时本地路径 Acquire 应成功")
	assert.True(t, g.IsRunning(12345))

	// 重复 Acquire 应失败（本地互斥）
	assert.False(t, g.Acquire(12345))

	// Release 应清理本地状态
	g.Release(12345)
	assert.False(t, g.IsRunning(12345))

	// 释放后可再次获取
	assert.True(t, g.Acquire(12345))
	g.Release(12345)
}

func TestRunGuard_Redis_LocalToRedisTransition(t *testing.T) {
	// 场景：先在本地模式 Acquire，然后恢复 Redis
	// 验证：本地 held 状态持续有效，Release 正确处理
	g := &RunGuard[uint64]{}
	key := "mayfly:run-guard:11111"
	rediscli.GetCli().Del(ctx, key)
	defer rediscli.GetCli().Del(ctx, key)

	cli := rediscli.GetCli()

	// 阶段1：Redis 不可用，本地模式 Acquire
	rediscli.SetCli(nil)
	require.True(t, g.Acquire(11111), "本地模式 Acquire 应成功")
	assert.True(t, g.IsRunning(11111))

	// 阶段2：恢复 Redis，本地 held 仍有效
	rediscli.SetCli(cli)
	assert.True(t, g.IsRunning(11111), "恢复 Redis 后本地 held 仍有效")
	assert.False(t, g.Acquire(11111), "重复 Acquire 仍应失败")

	// 阶段3：Release 应同时清理本地和 Redis（虽然 Redis 中无 key，CasDel 安全返回）
	g.Release(11111)
	assert.False(t, g.IsRunning(11111))
}

// ========== BindCronTask: 重新绑定替换回调 ==========

func TestBindCronTask_RebindReplacesCallback(t *testing.T) {
	key := "taskx_test_rebind"
	defer UnbindCronTask(key)

	var counter1, counter2 int32
	done1 := make(chan struct{}, 1)
	done2 := make(chan struct{}, 1)

	// 第一次绑定：快速触发，递增 counter1
	require.NoError(t, BindCronTask(key, "@every 100ms", true, func() {
		atomic.AddInt32(&counter1, 1)
		select {
		case done1 <- struct{}{}:
		default:
		}
	}))
	assert.True(t, scheduler.ExistKey(key))

	// 等待第一次回调触发
	<-done1

	// 重新绑定：替换为递增 counter2 的回调
	require.NoError(t, BindCronTask(key, "@every 100ms", true, func() {
		atomic.AddInt32(&counter2, 1)
		select {
		case done2 <- struct{}{}:
		default:
		}
	}))
	assert.True(t, scheduler.ExistKey(key))

	// 等待第二次回调触发
	select {
	case <-done2:
	case <-time.After(2 * time.Second):
		t.Fatal("重新绑定后回调应在 2s 内触发")
	}

	// 记录当前值
	c1AfterRebind := atomic.LoadInt32(&counter1)
	// 等一小段时间，counter1 不应再增长（旧回调已替换）
	time.Sleep(300 * time.Millisecond)
	c1Final := atomic.LoadInt32(&counter1)
	assert.Equal(t, c1AfterRebind, c1Final, "重新绑定后旧回调不应再执行")
	assert.True(t, atomic.LoadInt32(&counter2) > 0, "新回调应已执行")
}
