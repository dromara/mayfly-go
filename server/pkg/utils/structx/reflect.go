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


// IsZeroValue 检查字段是否为零值
func IsZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return v.IsNil()
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
}