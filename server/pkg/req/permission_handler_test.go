package req

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDefaultPermissionCodeRegistryLoadsOnMiss 进程内缓存未命中时必须回源加载并回填：
// 这是「后端重启后旧 token 仍有效、但权限缓存为空」不再被误判为无权限的关键。
func TestDefaultPermissionCodeRegistryLoadsOnMiss(t *testing.T) {
	var calls int32
	SetPermissionCodeLoader(func(userId uint64) ([]string, bool) {
		atomic.AddInt32(&calls, 1)
		if userId != 7 {
			return nil, false
		}
		return []string{"db:sqlscript:run"}, true
	})
	t.Cleanup(func() { SetPermissionCodeLoader(nil) })

	registry := new(DefaultPermissionCodeRegistry)
	require.True(t, registry.HasCode(7, "db:sqlscript:run"), "冷缓存必须回源加载")
	require.False(t, registry.HasCode(7, "account:del"), "已加载但无该 code 必须拒绝")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "第二次命中缓存，不得重复回源")
	require.False(t, registry.HasCode(8, "db:sqlscript:run"), "loader ok=false 必须 fail-closed")

	registry.Remove(7)
	require.True(t, registry.HasCode(7, "db:sqlscript:run"), "Remove 后必须重新回源")
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

// TestDefaultPermissionCodeRegistryCachesEmptyCodes 有效账号但无任何按钮权限时，
// 空权限集也必须回填，否则每个请求都会回源查库。
func TestDefaultPermissionCodeRegistryCachesEmptyCodes(t *testing.T) {
	var calls int32
	SetPermissionCodeLoader(func(uint64) ([]string, bool) {
		atomic.AddInt32(&calls, 1)
		return []string{}, true
	})
	t.Cleanup(func() { SetPermissionCodeLoader(nil) })

	registry := new(DefaultPermissionCodeRegistry)
	require.False(t, registry.HasCode(9, "db:sqlscript:run"))
	require.False(t, registry.HasCode(9, "db:sqlscript:run"))
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "空权限集也应命中缓存")
}

// TestDefaultPermissionCodeRegistryNoLoader 未注册 loader 时必须保持原语义：未命中即无权限。
func TestDefaultPermissionCodeRegistryNoLoader(t *testing.T) {
	SetPermissionCodeLoader(nil)
	registry := new(DefaultPermissionCodeRegistry)
	require.False(t, registry.HasCode(1, "any"))
}

// TestDefaultPermissionCodeRegistryConcurrentColdStart 并发冷启动只允许初始化一次缓存，
// 且不得出现 data race（配合 -race 运行）。
func TestDefaultPermissionCodeRegistryConcurrentColdStart(t *testing.T) {
	SetPermissionCodeLoader(func(uint64) ([]string, bool) { return []string{"a"}, true })
	t.Cleanup(func() { SetPermissionCodeLoader(nil) })

	registry := new(DefaultPermissionCodeRegistry)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !registry.HasCode(1, "a") {
				t.Error("并发冷启动下权限码必须命中")
			}
		}()
	}
	wg.Wait()
}
