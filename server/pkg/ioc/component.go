package ioc

import "reflect"

type ComponentOption func(component *Component)

// 组件名
func WithComponentName(name string) ComponentOption {
	return func(c *Component) {
		c.Name = name
	}
}

// 组件
type Component struct {
	Name string // 组件名

	Value any // 组件实例
}

// GetType 获取组件实例类型
func (c *Component) GetType() reflect.Type {
	return reflect.TypeOf(c.Value)
}

func NewComponent(val any, opts ...ComponentOption) *Component {
	component := &Component{
		Value: val,
	}

	for _, o := range opts {
		o(component)
	}
	return component
}
