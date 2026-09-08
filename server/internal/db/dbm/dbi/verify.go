package dbi

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CanonicalNilValue nil值的规范化表示
const CanonicalNilValue = "<nil>"

// CanonicalValue 将任意驱动返回值规范化为跨方言可比对的字符串。
//
// 数据校验需要对源库与目标库逐行逐列比对，但不同方言驱动对同一逻辑值的返回形态
// 存在差异（如时间时区/精度、二进制类型、数值类型宽窄），直接比较会产生大量误报。
// 规范化规则：
//   - nil → "<nil>"
//   - time.Time → RFC3339Nano（统一转UTC，消除时区差异）
//   - []byte → "0x" + 小写hex（二进制值统一形态，且与前缀区分于普通字符串）
//   - 各整型/浮点数值 → 统一十进制字符串（float用最短精确表示）
//   - bool → "true"/"false"
//   - string → 原样
//   - 其他未知形态 → fmt.Sprintf("%v")
func CanonicalValue(v any) string {
	switch x := v.(type) {
	case nil:
		return CanonicalNilValue
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	case []byte:
		return "0x" + hex.EncodeToString(x)
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(reflectInt64(x), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflectUint64(x), 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func reflectInt64(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	}
	return 0
}

func reflectUint64(v any) uint64 {
	switch x := v.(type) {
	case uint:
		return uint64(x)
	case uint8:
		return uint64(x)
	case uint16:
		return uint64(x)
	case uint32:
		return uint64(x)
	case uint64:
		return x
	}
	return 0
}

// CanonicalEqual 比较两个驱动值规范化后是否相等（严格形态比较，不做数值归一）
//
// NULL与非NULL先行区分：否则真实字符串值恰好为 CanonicalNilValue（"<nil>"）时，
// 会与另一侧的NULL误判为相等导致差异漏报
func CanonicalEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return CanonicalValue(a) == CanonicalValue(b)
}

// CanonicalNumericEqual 数值类列专用等值判断：先按规范化形态比较，
// 不等时再按纯十进制数值精确比较（big.Rat），消除同一数值在各方言间的标度呈现差异
// （如mysql/pg numeric(20,6)返回"123.456000"而sqlite NUMERIC亲和为"123.456"）导致的校验误报。
//
// 必须仅用于两侧列类型均为数值类的场景：对文本列使用会将"1"与"1.0"等真实差异误判为相等
func CanonicalNumericEqual(a, b any) bool {
	if CanonicalEqual(a, b) {
		return true
	}
	return decimalEqual(CanonicalValue(a), CanonicalValue(b))
}

// plainDecimalReg 纯十进制数字形态（可负、可带小数部分），不含分数/指数/十六进制等歧义形态
var plainDecimalReg = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// decimalEqual 两个纯十进制形态值是否数值相等（不丢精度）。
//
// plainDecimalReg 严格限定为可负、可带小数的纯数字形态，排除"1/2"、"1e3"、"0x01"等
// 歧义形态，避免不同文本被误归一为相等造成差异漏报
func decimalEqual(a, b string) bool {
	if !plainDecimalReg.MatchString(a) || !plainDecimalReg.MatchString(b) {
		return false
	}
	ra, ok1 := new(big.Rat).SetString(a)
	rb, ok2 := new(big.Rat).SetString(b)
	return ok1 && ok2 && ra.Cmp(rb) == 0
}

// IsNumericCommonType 公共列类型是否为数值类（位/布尔/各宽度整数/无符号整数/定点数）
func IsNumericCommonType(ct CommonDbDataType) bool {
	switch ct {
	case CTBit, CTBool, CTInt1, CTInt2, CTInt4, CTInt8, CTNumeric, CTDecimal,
		CTUnsignedInt8, CTUnsignedInt4, CTUnsignedInt2, CTUnsignedInt1:
		return true
	default:
		return false
	}
}

// canonicalPart 拼接比对用的单值形态：NULL使用仅NULL可达的前缀形态，
// 与任何真实字符串值不可能碰撞
func canonicalPart(v any) string {
	if v == nil {
		return "N:"
	}
	return "V:" + CanonicalValue(v)
}

// CanonicalRowKey 生成行指定列的规范化键（用于主键对齐查找）
func CanonicalRowKey(row map[string]any, column string) string {
	return CanonicalValue(row[column])
}

// CanonicalRowSignature 生成整行的规范化签名（按传入列顺序拼接），用于整行比对
//
// 列名/列值以控制字符分隔且值带类型前缀：避免值中含 , = 等分隔符时两行拼接出相同签名而漏报差异
func CanonicalRowSignature(row map[string]any, columns []string) string {
	parts := make([]string, 0, len(columns))
	for _, c := range columns {
		parts = append(parts, c+"\x1f"+canonicalPart(row[c]))
	}
	return strings.Join(parts, "\x1e")
}
