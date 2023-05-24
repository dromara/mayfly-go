package utils

import (
	"strconv"
)

// any类型转换为string, 如果any为nil则返回空字符串
func Any2String(val any) string {
	if value, ok := val.(string); !ok {
		return ""
	} else {
		return value
	}
}

// any类型转换为int（可将字符串或int64转换）, 如果any为nil则返回0
func Any2Int(val any) int {
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
func Any2Int64(val any) int64 {
	if value, ok := val.(int64); !ok {
		return int64(Any2Int(val))
	} else {
		return value
	}
}
