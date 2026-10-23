package structx

import "reflect"

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
	case reflect.Pointer, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return v.IsNil()
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
}

// newInstance 创建一个 T 类型的实例
// 如果 T 是指针类型 (*A)，它返回 new(A) 并转换为 *A
// 如果 T 是值类型 (A)，它返回 new(A) 的解引用值 A (即零值)，或者根据需求调整
func NewInstance[T any]() T {
	var t T
	typ := reflect.TypeOf(t)

	if typ.Kind() == reflect.Pointer {
		// T 是 *SomeStruct
		// 我们需要创建 SomeStruct 的实例，并返回其指针
		elemType := typ.Elem()
		v := reflect.New(elemType)
		return v.Interface().(T)
	} else {
		// T 是 SomeStruct
		// 返回零值结构体，或者返回 &SomeStruct{} (但这会改变类型)
		return t
	}
}
