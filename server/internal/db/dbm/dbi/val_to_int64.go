package dbi

import (
	"strconv"
	"strings"
)

// ValToFloat64 将驱动返回的数值型值转为float64。
// REAL/DOUBLE/FLOAT列的返值形态：float64/int64/[]byte/string
func ValToFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case nil:
		return 0, false
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int64:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case []byte:
		return ParseStrToFloat64(string(x))
	case string:
		return ParseStrToFloat64(x)
	default:
		return 0, false
	}
}

func ParseStrToFloat64(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// ValToBool 将驱动返回的逻辑值转为bool。
// bool列返值形态：bool/int64（mysql tinyint(1)）/string（"true"/"false"/"t"/"f"/"1"/"0"）
func ValToBool(v any) (bool, bool) {
	switch x := v.(type) {
	case nil:
		return false, false
	case bool:
		return x, true
	case int64:
		return x != 0, true
	case int:
		return x != 0, true
	case int8:
		return x != 0, true
	case int16:
		return x != 0, true
	case int32:
		return x != 0, true
	case uint64:
		return x != 0, true
	case uint8:
		return x != 0, true
	case uint16:
		return x != 0, true
	case uint32:
		return x != 0, true
	case float32:
		return x != 0, true
	case float64:
		return x != 0, true
	case []byte:
		return ParseStrToBool(string(x))
	case string:
		return ParseStrToBool(x)
	default:
		return false, false
	}
}

func ParseStrToBool(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "t", "1", "yes", "y", "on":
		return true, true
	case "false", "f", "0", "no", "n", "off":
		return false, true
	}
	return false, false
}

// ValToInt64 将驱动返回的数值型值转为int64。
// 不同方言驱动对MIN/MAX(int列)的返回形态不一：int64/int/float64/[]byte/string等
func ValToInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case nil:
		return 0, false
	case int64:
		return x, true
	case int:
		return int64(x), true
	case int32:
		return int64(x), true
	case uint64:
		return int64(x), true
	case float64:
		return int64(x), true
	case []byte:
		return ParseStrToInt64(string(x))
	case string:
		return ParseStrToInt64(x)
	default:
		return 0, false
	}
}

func ParseStrToInt64(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// float形态字符串（如"1.0"）
		if f, ferr := strconv.ParseFloat(s, 64); ferr == nil {
			return int64(f), true
		}
		return 0, false
	}
	return n, true
}
