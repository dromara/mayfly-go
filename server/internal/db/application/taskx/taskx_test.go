package taskx

import (
	"sync"
	"testing"

	"mayfly-go/pkg/scheduler"

	"github.com/stretchr/testify/assert"
)

// TestRunGuard_AcquireRelease 基本的获取/释放语义
func TestRunGuard_AcquireRelease(t *testing.T) {
	g := &RunGuard{}
	assert.False(t, g.IsRunning(1), "初始状态不应为运行中")

	assert.True(t, g.Acquire(1), "首次获取应成功")
	assert.True(t, g.IsRunning(1))

	assert.False(t, g.Acquire(1), "运行中重复获取应失败")
	assert.False(t, g.Acquire(1), "失败后仍不应改变运行态")

	g.Release(1)
	assert.False(t, g.IsRunning(1))
	assert.True(t, g.Acquire(1), "释放后可再次获取")

	// 重复Release安全
	g.Release(1)
	g.Release(1)
}

// TestRunGuard_TaskIsolation 不同任务互不影响
func TestRunGuard_TaskIsolation(t *testing.T) {
	g := &RunGuard{}
	assert.True(t, g.Acquire(1))
	assert.True(t, g.Acquire(2), "任务2不受任务1运行态影响")
	assert.True(t, g.IsRunning(1))
	assert.True(t, g.IsRunning(2))
	g.Release(1)
	assert.False(t, g.IsRunning(1))
	assert.True(t, g.IsRunning(2))
}

// TestRunGuard_ConcurrentAcquire 并发获取同一任务仅一个成功（竞态修复的核心语义）
func TestRunGuard_ConcurrentAcquire(t *testing.T) {
	g := &RunGuard{}
	const workers = 64

	var acquired int32
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if g.Acquire(42) {
				mu.Lock()
				acquired++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), acquired, "并发Acquire仅应一个成功")
	assert.True(t, g.IsRunning(42))
}

// TestBindCronTask 禁用路径：仅移除旧绑定，恒成功
func TestBindCronTask_Disabled(t *testing.T) {
	key := "taskx_test_disabled"
	assert.NoError(t, BindCronTask(key, "* * * * * *", false, func() {}))
	assert.False(t, scheduler.ExistKey(key))
}

// TestBindCronTask_InvalidSpec 启用路径：非法cron表达式返回错误且不注册
func TestBindCronTask_InvalidSpec(t *testing.T) {
	key := "taskx_test_invalid"
	assert.Error(t, BindCronTask(key, "not-a-cron", true, func() {}))
	assert.False(t, scheduler.ExistKey(key))
}

// TestBindCronTask_BindUnbind 正常绑定/解绑
func TestBindCronTask_BindUnbind(t *testing.T) {
	key := "taskx_test_bind"
	assert.NoError(t, BindCronTask(key, "@every 1h", true, func() {}))
	assert.True(t, scheduler.ExistKey(key))

	// 重新绑定：先移除旧任务再注册
	assert.NoError(t, BindCronTask(key, "@every 2h", true, func() {}))
	assert.True(t, scheduler.ExistKey(key))

	UnbindCronTask(key)
	assert.False(t, scheduler.ExistKey(key))

	// 解绑不存在的key安全
	UnbindCronTask(key)
}
