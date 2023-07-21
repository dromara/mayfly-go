package anyx

import (
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
