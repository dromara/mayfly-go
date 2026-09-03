package ioc

import ()

// 全局默认实例容器
var DefaultContainer = NewContainer()

// 注册实例至全局默认ioc容器
func Register(component any, opts ...ComponentOption) {
	DefaultContainer.Register(component, opts...)
}

// RegisterByType 根据组件实例类型注册至全局默认ioc容器，会自动创建实例
func RegisterByType[T any](opts ...ComponentOption) {
	DefaultContainer.RegisterByType[T](opts...)
}

// Get 根据组件实例类型从全局默认ioc容器获取实例
func Get[T any]() T {
	return DefaultContainer.GetBean[T]()
}

// GetByName 根据组件名从全局默认ioc容器获取实例
func GetByName[T any](name string) T {
	return DefaultContainer.GetBeanByName[T](name)
}

// GetBeansByType 根据组件实例类型从全局默认ioc容器获取实例
func GetBeansByType[T any]() []T {
	return DefaultContainer.GetBeans[T]()
}

// Inject 使用全局默认ioc容器中已注册的组件实例 -> 注入到指定实例所依赖的组件实例
func Inject(component any) error {
	return DefaultContainer.Inject(component)
}

// InjectComponents 注入默认ioc容器内组件所依赖的其他组件实例
func InjectComponents() error {
	return DefaultContainer.InjectComponents()
}
