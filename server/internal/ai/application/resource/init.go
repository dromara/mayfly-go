package resource

import "sync"

// 默认 App（进程级单例）：由 ai/application.Init 启动期装配，
// 未装配时 GetApp 返回 nil，调用方需判空（fail-open）。
var (
	defaultMu  sync.RWMutex
	defaultApp App
)

// Init 装配默认资源查询 App 并注册内置 provider（启动期调用一次）
func Init() {
	app := NewApp()
	app.RegisterProvider(&machineProvider{})
	app.RegisterProvider(&dbProvider{})
	SetDefault(app)
}

// SetDefault 替换默认 App（测试注入扩展点）
func SetDefault(app App) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultApp = app
}

// GetApp 获取默认 App；未装配时返回 nil
func GetApp() App {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultApp
}
