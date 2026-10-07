package trigger

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// toFloat 尽力把任意标量转为浮点数，用于数值比较。
// JSON 反序列化后整数会变成 float64、表单可能提交字符串数字，因此需要统一归化
func toFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case nil:
		return 0, false
	case bool:
		return 0, false
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(reflected.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(reflected.Uint()), true
	case reflect.Pointer, reflect.Interface:
		if reflected.IsNil() {
			return 0, false
		}
		return toFloat(reflected.Elem().Interface())
	}
	return 0, false
}

// toBool 判定布尔取值，字符串 "true"/"1" 视为真（表单常以字符串提交）
func toBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return err == nil && parsed
	case nil:
		return false
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Bool:
		return reflected.Bool()
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return reflected.Len() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflected.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflected.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return reflected.Float() != 0
	}
	return false
}

// asText 取值的原始文本形态，用于文本包含/前缀/正则比较。
// 不能复用 canonical：它会把 "007" 归一成 "7"、"10.0" 归一成 "10"，
// 于是「SQL 包含 007」会命中只含 7 的语句，文本匹配语义被数值归一污染
func asText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	case bool:
		return strconv.FormatBool(typed)
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface {
		if reflected.IsNil() {
			return ""
		}
		return asText(reflected.Elem().Interface())
	}
	// reflect.Value.String() 对非字符串 Kind 返回的是「<int Value>」这类占位串，
	// 会让「包含 007」之类的文本判定无声地永远不成立，故取值的常规文本形态用 fmt
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

// canonical 生成用于相等比较的规范字符串：数值去掉多余小数位，避免 3 与 3.0 判不等
func canonical(value any) string {
	if number, ok := toFloat(value); ok {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	switch typed := value.(type) {
	case nil:
		return ""
	case bool:
		return strconv.FormatBool(typed)
	case string:
		return typed
	case []byte:
		return string(typed)
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface {
		if reflected.IsNil() {
			return ""
		}
		return canonical(reflected.Elem().Interface())
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

// looseEq 宽松相等：两侧都能归化为数值时按数值比较，否则按规范字符串比较。
//
// 任一侧为布尔时按布尔归一后比较：字段字典的原始输入恒为字符串（map[string]string），
// 若要求两侧都是 bool 类型，布尔字段的条件将永远不成立。
// 数值与布尔互认（1 == true）由保存校验挡住：布尔字段的期望值必须是真布尔
func looseEq(actual, expected any) bool {
	if isBool(actual) || isBool(expected) {
		return toBool(actual) == toBool(expected)
	}
	actualNumber, actualOk := toFloat(actual)
	expectedNumber, expectedOk := toFloat(expected)
	if actualOk && expectedOk {
		return actualNumber == expectedNumber
	}
	return strings.EqualFold(canonical(actual), canonical(expected))
}

func isBool(value any) bool {
	if _, ok := value.(bool); ok {
		return true
	}
	reflected := reflect.ValueOf(value)
	return reflected.Kind() == reflect.Bool
}

// toList 把任意取值展开为元素列表，非列表类型展开为单元素列表，nil 展开为空列表
func toList(value any) []any {
	if value == nil {
		return nil
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Slice, reflect.Array:
		items := make([]any, 0, reflected.Len())
		for index := 0; index < reflected.Len(); index++ {
			items = append(items, reflected.Index(index).Interface())
		}
		return items
	case reflect.Map:
		keys := reflected.MapKeys()
		items := make([]any, 0, len(keys))
		for _, key := range keys {
			items = append(items, key.Interface())
		}
		return items
	}
	return []any{value}
}

// isEmptyValue 空值判定：nil、空串、空集合、空 map 视为空。数值 0 与 false 不算空
func isEmptyValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return reflected.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return reflected.IsNil()
	}
	return false
}

// splitRange 把区间型期望值拆为左右端点，兼容 [min,max] 数组与 {min,max} 对象两种下发形态
func splitRange(expected any) (any, any, bool) {
	reflected := reflect.ValueOf(expected)
	switch reflected.Kind() {
	case reflect.Slice, reflect.Array:
		if reflected.Len() != 2 {
			return nil, nil, false
		}
		return reflected.Index(0).Interface(), reflected.Index(1).Interface(), true
	case reflect.Map:
		lower := mapValue(expected, "min", "from", "start")
		upper := mapValue(expected, "max", "to", "end")
		if lower == nil || upper == nil {
			return nil, nil, false
		}
		return lower, upper, true
	}
	return nil, nil, false
}

// ParamStrings 读取字符串列表型参数，供检查项实现复用（参数已经过保存时校验，此处只做形态归化）
func ParamStrings(params map[string]any, key string) []string {
	items := toList(params[key])
	values := make([]string, 0, len(items))
	for _, item := range items {
		if text := asText(item); text != "" {
			values = append(values, text)
		}
	}
	return values
}

// ParamFloat 读取数值型参数，缺失或不可归化时返回兜底值
func ParamFloat(params map[string]any, key string, fallback float64) float64 {
	number, ok := toFloat(params[key])
	if !ok {
		return fallback
	}
	return number
}

// ParamBool 读取布尔型参数
func ParamBool(params map[string]any, key string) bool {
	return toBool(params[key])
}

// ContainsFold 忽略大小写的包含判定，适用于 SQL 关键字与命令名
func ContainsFold(text, keyword string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(keyword))
}

// OneOf 判定取值是否在候选集内（忽略大小写）
func OneOf(value string, candidates []string) bool {
	for _, candidate := range candidates {
		if looseEq(value, candidate) {
			return true
		}
	}
	return false
}

// mapValue 按键名读取字符串键 map。非字符串键直接返回 nil，避免 reflect 因键类型不匹配 panic
func mapValue(container any, keys ...string) any {
	reflected := reflect.ValueOf(container)
	if reflected.Kind() != reflect.Map || reflected.Type().Key().Kind() != reflect.String {
		return nil
	}
	for _, key := range keys {
		value := reflected.MapIndex(reflect.ValueOf(key))
		if value.IsValid() {
			return value.Interface()
		}
	}
	return nil
}
