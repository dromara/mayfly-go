package ioc

// 全局默认实例容器
var DefaultContainer = NewContainer()

// 注册实例至全局默认ioc容器
func Register(component any, opts ...ComponentOption) {
	DefaultContainer.Register(component, opts...)
}

// 根据组件名从全局默认ioc容器获取实例
func Get[T any](name string) T {
	c, _ := DefaultContainer.Get(name)
	return c.(T)
}

// 使用全局默认ioc容器中已注册的组件实例 -> 注入到指定实例所依赖的组件实例
func Inject(component any) error {
	return DefaultContainer.Inject(component)
}

// 注入默认ioc容器内组件所依赖的其他组件实例
func InjectComponents() error {
	return DefaultContainer.InjectComponents()
}
