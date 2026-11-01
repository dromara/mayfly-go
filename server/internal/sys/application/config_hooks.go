package application

import (
	"context"
	"slices"
	"sync"

	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
)

// ConfigChangeHook 系统配置变更钩子（配置保存成功后异步触发）
//
// 参数 key 为发生变更的配置键，钩子内部按需重新读取配置值（此时缓存已失效，
// GetConfig 读到的就是新值）。
type ConfigChangeHook func(ctx context.Context, key string)

var (
	configChangeHookMu sync.RWMutex
	configChangeHooks  = make(map[string][]ConfigChangeHook)
)

// RegisterConfigChangeHook 注册指定配置键的变更钩子（同 key 可注册多个，按注册顺序触发）
//
// 用途：让把系统配置读进运行时的上层模块（如 AI 模型配置 → ChatModel 实例）
// 在配置保存后刷新自己的运行时，而 sys 包无需感知具体模块（依赖倒置、开闭原则：
// 新增一类需要热生效的配置，只在上层模块注册钩子，本包零改动）。
//
// 钩子为异步执行：保存接口的返回不等待钩子，钩子失败 / panic 也不会回滚或阻塞保存
// （配置已落库，刷新失败只影响本轮运行期行为，日志留痕后由下次保存或重启兜底）。
func RegisterConfigChangeHook(key string, hook ConfigChangeHook) {
	if key == "" || hook == nil {
		return
	}
	configChangeHookMu.Lock()
	defer configChangeHookMu.Unlock()
	configChangeHooks[key] = append(configChangeHooks[key], hook)
}

// notifyConfigChanged 异步通知配置变更（key 精确匹配，无钩子时零开销直接返回）
func notifyConfigChanged(ctx context.Context, key string) {
	configChangeHookMu.RLock()
	hooks := slices.Clone(configChangeHooks[key])
	configChangeHookMu.RUnlock()
	if len(hooks) == 0 {
		return
	}

	// 请求 ctx 在响应写回后会被取消，钩子需要独立生命周期：只继承链路 ID，
	// 不继承取消/超时（否则「保存成功后刷新运行时」会被拆链路取消掉）
	hookCtx := contextx.WithTraceId(context.WithoutCancel(ctx))

	gox.Go(func() {
		for _, hook := range hooks {
			// 逐个隔离 panic：一个模块的刷新失败不得影响其它订阅者
			func() {
				defer gox.Recover()
				hook(hookCtx, key)
			}()
		}
	}, func(err error) {
		logx.ErrorfContext(hookCtx, "[sys-config] change hook for key [%s] panicked: %v", key, err)
	})
}
