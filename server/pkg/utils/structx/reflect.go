package structx

import "reflect"

// NewInstance 创建泛型 T 的实例。如果 T 是指针类型，则创建其指向类型的实例并返回指针。
func NewInstance[T any]() T {
	var t T

	// 反射判断是否是指针类型，并且是否为 nil
	if reflect.ValueOf(t).Kind() == reflect.Ptr {
		// 创建 T 对应的非指针类型的实例，并取其地址作为新的 T
		t = reflect.New(reflect.TypeOf(t).Elem()).Interface().(T)
	} else if kind := reflect.TypeOf(t).Kind(); kind == reflect.Array || kind == reflect.Slice {
		// 如果是数组或切片类型，创建一个新的切片（数组）
		elemType := reflect.TypeOf(t).Elem()
		newSlice := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)
		t = newSlice.Interface().(T)
	}

	return t
}
