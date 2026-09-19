package dbi

import (
	"fmt"

	"github.com/spf13/cast"
)

// DataType 数据类型, 对应于go类型，如int int64等。可自定义其他类型
type DataType struct {
	Name string //  类型名

	Valuer func() Valuer // 获取值对应的处理者，用于sql的scan、解析value等

	SQLValue func(val any) string // 转换为sql字符串值，用于insert等SQL语句的值转换
}

// Copy 拷贝一个同类型的datatype，主要方便用于定制化修改Valuer或ToString
func (dt *DataType) Copy() *DataType {
	return &DataType{
		Name:     dt.Name,
		Valuer:   dt.Valuer,
		SQLValue: dt.SQLValue,
	}
}

func (dt *DataType) WithValuer(valuerFunc func() Valuer) *DataType {
	dt.Valuer = valuerFunc
	return dt
}

func (dt *DataType) WithSQLValue(sqlvalueFunc func(val any) string) *DataType {
	dt.SQLValue = sqlvalueFunc
	return dt
}

const NULL = "NULL"

// SQLValue 序列化函数族——扩展模型与命名约定（后续维护者请先读这段）：
//
// 本族每个函数是一种**独立的转义/字面量策略，按机制命名而非按方言命名**。方言不在本文件
// 分支选择策略，而是在自家 column.go 经 DataType 槽位组合绑定：
//
//	DTStringMysql = dbi.DTString.Copy().WithSQLValue(dbi.SQLValueStringEscapeBackslash)
//
// 即 DataType{Valuer, SQLValue} 函数槽就是方言的覆写点（组合优于继承：方言不子类化，
// 按槽位绑定策略函数），本文件内保证零 dbType 分支。
// 注释中出现的方言名（mysql/pg/oracle/达梦等）是各策略如此设计的**实测证据**——
// 记录"为什么不能按另一种方式转义"，防止后续好心改坏（如对 pg 数据加反斜杠转义即成脏数据）。
// 方言专有怪癖策略若仅单一消费方应下沉到该方言包；出现 ≥2 个消费方才升入本词汇表
// （跨方言 import 是禁区）。当前例：SQLValueStringEscapeBackslash 由 mysql+clickhouse 共用。

// stringifyValue 将任意扫描值归一为字符串文本：[]byte取字节内容（而非%v打印出的字节数组），
// 其余类型使用fmt格式化
func stringifyValue(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// SQLValueDefault 日期/时间等类型转SQL字面量。
// 值可能为驱动原样透传的库内文本（如sqlite弱类型列、异构迁移脏数据），必须按SQL标准转义单引号，
// 否则会破坏字符串字面量甚至产生SQL注入
func SQLValueDefault(val any) string {
	if val == nil {
		return NULL
	}
	return fmt.Sprintf("'%s'", QuoteEscape(stringifyValue(val)))
}

// IsNumericLiteral 判断文本是否为可直接无引号嵌入SQL的数字字面量（十进制整数/小数/科学计数法，可带正负号）。
// 刻意不接受Inf/NaN/十六进制/下划线等strconv.ParseFloat可接受的写法，它们不是合法SQL字面量；
// 小数点后与指数符号后也必须至少有一位数字（如 1. 、1e 在mysql下为语法错误）
func IsNumericLiteral(s string) bool {
	i := 0
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		i = 1
	}
	var digits int
	var dotDigits int
	var dot, exp bool
	for ; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			digits++
			if dot && !exp {
				dotDigits++
			}
		case c == '.' && !dot && !exp:
			dot = true
		case (c == 'e' || c == 'E') && !exp && digits > 0:
			exp = true
			// 指数部分可带一个符号，且必须至少有一位数字
			if i+1 < len(s) && (s[i+1] == '+' || s[i+1] == '-') {
				i++
			}
			if i+1 >= len(s) {
				return false
			}
		default:
			return false
		}
	}
	return digits > 0 && (!dot || dotDigits > 0)
}

// SQLValueNumeric 数字类型转string。
// numeric/decimal等列的Valuer为字符串，库内可能存有非法文本（弱类型库、历史脏数据），
// 直接裸拼会产生语法错误甚至注入（如"1; DROP TABLE t--"），故仅合法数字字面量才无引号输出，
// 否则退化为转义字符串字面量交由目标库做类型校验（显式报错优于静默执行拼接出的SQL）
func SQLValueNumeric(val any) string {
	if val == nil {
		return NULL
	}
	strVal := stringifyValue(val)
	// 源方言布尔列（如pg boolean）经DTString读回为"true"/"false"文本，目标为tinyint(1)等
	// 数值语义列时必须输出布尔字面量而非字符串字面量：'true'写入mysql数值列报1366，
	// 而true/false字面量在mysql tinyint与pg/sqlite的bool/int列均合法（异构迁移实测抓出）
	if strVal == "true" || strVal == "false" {
		return strVal
	}
	if IsNumericLiteral(strVal) {
		return strVal
	}
	return fmt.Sprintf("'%s'", QuoteEscape(strVal))
}

// SQLValueBool 布尔值转SQL字面量
func SQLValueBool(val any) string {
	// 与其它SQLValue*保持一致：nil必须输出NULL而非false，
	// 否则布尔列的NULL在导出/迁移链路中被静默改写为false（数据失真）
	if val == nil {
		return NULL
	}
	return fmt.Sprintf("%v", cast.ToBool(val))
}

// SQLValueString 转换为SQL字符串值（标准SQL转义规则）
// 仅将单引号转义为两个单引号（SQL标准转义方式），其余字符原样保留：
// 反斜杠、双引号、换行符等在标准SQL字符串字面量中均为普通字符，
// 若对其做反斜杠转义（如 strconv.Quote），在 PostgreSQL/Oracle/达梦等数据库中会造成数据污染。
func SQLValueString(val any) string {
	if val == nil {
		return NULL
	}

	// 非string输入（如[]byte、数值、time.Time）也必须是带引号的字面量，
	// 直接%v输出会丢失引号（[]byte会打印为字节数组）导致语法错误与数据损坏
	return fmt.Sprintf("'%s'", QuoteEscape(stringifyValue(val)))
}

// SQLValueStringEscapeBackslash 转换为SQL字符串值（MySQL转义规则）
// MySQL 默认模式下反斜杠是转义字符（如 \\b 会被解释为退格符），
// 因此除单引号外还需将反斜杠转义为双反斜杠，否则含反斜杠的数据会静默损坏。
// 注意：仅适用于 MySQL/MariaDB 等默认启用反斜杠转义的数据库
func SQLValueStringEscapeBackslash(val any) string {
	if val == nil {
		return NULL
	}

	// 先转义反斜杠，再转义单引号
	escapedStr := QuoteEscapeBackslash(stringifyValue(val))
	return fmt.Sprintf("'%s'", escapedStr)
}

// SQLValuePreserveSpecialChars 转换为SQL字符串值，保留特殊字符如双引号、换行符等
// 仅转义单引号为两个单引号（SQL标准转义方式）
func SQLValuePreserveSpecialChars(val any) string {
	return SQLValueString(val)
}

// IsHexString 判断字符串是否为合法的十六进制编码字符串（非空、偶数长度且均为hex字符）。
// 迁移链路中二进制数据经ValuerBytes回读即为hex编码字符串
func IsHexString(s string) bool {
	if len(s) == 0 || len(s)%2 != 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// SQLValueBytes 二进制值转SQL字面量。
// 迁移链路中二进制数据经ValuerBytes回读为hex编码字符串，输出标准十六进制字面量X'...'
// 以保真还原二进制（MySQL/SQLite/Oracle/达梦均支持），
// 否则会将hex文本作为普通字符串写入blob导致数据失真；
// 非hex值（调用方直接构造的字符串）退化为字符串字面量转义处理
func SQLValueBytes(val any) string {
	if val == nil {
		return NULL
	}
	if strVal, ok := val.(string); ok && IsHexString(strVal) {
		return fmt.Sprintf("X'%s'", strVal)
	}
	return SQLValueString(val)
}

// 内置数据类型全集：Valuer（扫描读取）与 SQLValue（字面量写出）的成对组合，
// 各方言的类型清单（dialect/column.go）从这里取用并组合定制
var (
	DTBit = &DataType{
		Name:     "bit",
		Valuer:   ValuerBit,
		SQLValue: SQLValueNumeric,
	}

	DTBool = &DataType{
		Name:     "bool",
		Valuer:   ValuerBit,
		SQLValue: SQLValueBool,
	}

	DTByte = &DataType{
		Name:     "uint8",
		Valuer:   ValuerByte,
		SQLValue: SQLValueNumeric,
	}

	DTInt8 = &DataType{
		Name:     "int8",
		Valuer:   ValuerInt16,
		SQLValue: SQLValueNumeric,
	}

	DTInt16 = &DataType{
		Name:     "int16",
		Valuer:   ValuerInt16,
		SQLValue: SQLValueNumeric,
	}

	DTInt32 = &DataType{
		Name:     "int32",
		Valuer:   ValuerInt32,
		SQLValue: SQLValueNumeric,
	}

	DTInt64 = &DataType{
		Name:     "int64",
		Valuer:   ValuerInt64,
		SQLValue: SQLValueNumeric,
	}

	// 所有无符号类型，都使用int64存储
	DTUint64 = &DataType{
		Name:     "uint64",
		Valuer:   ValuerUint64,
		SQLValue: SQLValueNumeric,
	}

	// 使用string进行转换，避免长度过长导致精度丢失等
	DTNumeric = &DataType{
		Name:     "numeric",
		Valuer:   ValuerString,
		SQLValue: SQLValueNumeric,
	}

	DTDecimal = &DataType{
		Name:     "decimal",
		Valuer:   ValuerString,
		SQLValue: SQLValueNumeric,
	}

	DTString = &DataType{
		Name:     "string",
		Valuer:   ValuerString,
		SQLValue: SQLValueString,
	}

	// DTStringPreserveSpecial 用于需要保留双引号换行符等特殊字符的字符串类型
	DTStringPreserveSpecial = &DataType{
		Name:     "string",
		Valuer:   ValuerString,
		SQLValue: SQLValuePreserveSpecialChars,
	}

	DTDate = &DataType{
		Name:     "date",
		Valuer:   ValuerDate,
		SQLValue: SQLValueDefault,
	}

	DTTime = &DataType{
		Name:     "time",
		Valuer:   ValuerTime,
		SQLValue: SQLValueDefault,
	}

	DTDateTime = &DataType{
		Name:     "datetime",
		Valuer:   ValuerDatetime,
		SQLValue: SQLValueDefault,
	}

	DTBytes = &DataType{
		Name:     "bytes",
		Valuer:   ValuerBytes,
		SQLValue: SQLValueBytes,
	}
)
