package anyx

import (
	"encoding/json"
	"math"
	"reflect"
	"strconv"
)

// any类型转换为string, 如果any为nil则返回空字符串
func ConvString(val any) string {
	if value, ok := val.(string); !ok {
		return ""
	} else {
		return value
	}
}

// any类型转换为int（可将字符串或int64转换）, 如果any为nil则返回0
func ConvInt(val any) int {
	switch value := val.(type) {
	case int:
		return value
	case string:
		if intV, err := strconv.Atoi(value); err == nil {
			return intV
		}
	case int64:
		return int(value)
	case uint64:
		return int(value)
	case int32:
		return int(value)
	case uint32:
		return int(value)
	case int16:
		return int(value)
	case uint16:
		return int(value)
	case int8:
		return int(value)
	case uint8:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(math.Round(value))
	default:
		return 0
	}
	return 0
}

// any类型转换为int64, 如果any为nil则返回0
func ConvInt64(val any) int64 {
	if value, ok := val.(int64); !ok {
		return int64(ConvInt(val))
	} else {
		return value
	}
}

func IsBlank(value any) bool {
	if value == nil {
		return true
	}
	rValue := reflect.ValueOf(value)
	switch rValue.Kind() {
	case reflect.String:
		return rValue.Len() == 0
	case reflect.Bool:
		return !rValue.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rValue.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rValue.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rValue.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return rValue.IsNil()
	}
	return reflect.DeepEqual(rValue.Interface(), reflect.Zero(rValue.Type()).Interface())
}

// any to string
func ToString(value any) string {
	// interface 转 string
	if value == nil {
		return ""
	}

	switch it := value.(type) {
	case string:
		return it
	case error:
		return it.Error()
	case float64:
		return strconv.FormatFloat(it, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(it), 'f', -1, 64)
	case int:
		return strconv.Itoa(it)
	case uint:
		return strconv.Itoa(int(it))
	case int8:
		return strconv.Itoa(int(it))
	case uint8:
		return strconv.Itoa(int(it))
	case int16:
		return strconv.Itoa(int(it))
	case uint16:
		return strconv.Itoa(int(it))
	case int32:
		return strconv.Itoa(int(it))
	case uint32:
		return strconv.Itoa(int(it))
	case int64:
		return strconv.FormatInt(it, 10)
	case uint64:
		return strconv.FormatUint(it, 10)
	case []byte:
		return string(it)
	default:
		newValue, _ := json.Marshal(value)
		return string(newValue)
	}
}

// DeepZero 初始化对象
// 如 T 为基本类型或结构体，则返回零值
// 如 T 为指向基本类型或结构体的指针，则返回指向零值的指针
func DeepZero[T any]() T {
	var data T
	typ := reflect.TypeOf(data)
	kind := typ.Kind()
	if kind == reflect.Pointer {
		return reflect.New(typ.Elem()).Interface().(T)
	}
	return data
}
