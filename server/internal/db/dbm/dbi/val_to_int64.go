package dbi

import (
	"math"
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
// 不同方言驱动对MIN/MAX(int列)的返回形态不一：int64/int/float64/[]byte/string等。
// 整型族各宽度必须逐一覆盖，不能只按int64/int假设：本机实测 mysql/pg/mssql 对
// SMALLINT 列的 MIN/MAX 均返回 int16（mssql：TINYINT/SMALLINT→int16、INT→int32、BIGINT→int64、COUNT→int32）；
// 漏掉窄整型会让 smallint 主键表的分片规划静默退化为整表单任务迁移（并行度丢失且无任何日志）
func ValToInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case nil:
		return 0, false
	case int64:
		return x, true
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case uint:
		return int64(x), true
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		return int64(x), true
	case float32:
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

// maxInt64Float 为2^63，int64可表示的最大值+1；绝对值达到该量级的浮点已不可安全转为int64
const maxInt64Float = 9223372036854775808.0

func ParseStrToInt64(s string) (int64, bool) {
	// 定宽文本/部分驱动返回的数值串带首尾空格，与ParseStrToFloat64保持同样的裁剪口径
	s = strings.TrimSpace(s)
	n, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return n, true
	}
	// float形态字符串（如"1.0"）：仅当文本带小数点或指数记号时才走浮点兜底，
	// 避免把下划线字面量（"1_000"）、inf/nan、超出int64范围的无符号大整数等
	// 非预期文本误当数值接受；float64→int64越界转换结果未定义，静默污染分片边界比取不到值更危险
	if strings.ContainsAny(s, ".eE") {
		if f, ferr := strconv.ParseFloat(s, 64); ferr == nil && !math.IsInf(f, 0) && !math.IsNaN(f) && math.Abs(f) < maxInt64Float {
			return int64(f), true
		}
	}
	return 0, false
}
