package enumx

import (
	"fmt"
	"mayfly-go/pkg/errorx"
)

type Enum[T comparable] struct {
	name string // 枚举值名称

	values map[T]string // 所有枚举值。枚举值 -> desc(描述)
}

// 新建枚举
func NewEnum[T comparable](name string) *Enum[T] {
	return &Enum[T]{
		name:   name,
		values: make(map[T]string),
	}
}

// 添加枚举值
func (e *Enum[T]) Add(value T, desc string) *Enum[T] {
	e.values[value] = desc
	return e
}

// 校验枚举值是否合法
func (e *Enum[T]) Valid(value T) error {
	_, ok := e.values[value]
	if ok {
		return nil
	}

	errMsg := fmt.Sprintf("%s可选值为: ", e.name)
	for val, desc := range e.values {
		errMsg = fmt.Sprintf("%s [%v->%s]", errMsg, val, desc)
	}
	return errorx.NewBiz(errMsg)
}

// 根据枚举值获取描述
func (e *Enum[T]) GetDesc(value T) string {
	desc, ok := e.values[value]
	if ok {
		return desc
	}

	return ""
}
