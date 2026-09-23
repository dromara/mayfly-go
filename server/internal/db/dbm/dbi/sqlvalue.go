package dbi

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/spf13/cast"
)

// ValueCodec 值编解码策略描述符：绑定一个 Go 值语义（如 int/int64/string）到三个函数槽：
//   - Valuer：扫描读取（sql.Rows 结果 → 中间值）；
//   - SQLValue：写入字面量（中间值 → 拼入 SQL 文本的完整字面量）；
//   - BindValue：写入绑定值（中间值 → 驱动参数化接口可接受的 Go 值）。
//
// SQLValue 与 BindValue 描述同一写入语义的两条实现路径，同一目标列不管走哪条都必须落同一结果；
// 新方言或新中间值形态时两边同步实现，不得只在应用层重新列枚分珥。方言按槽位组合出 DT* 常量，
// 可自定义其他类型。
type ValueCodec struct {
	Name string //  类型名

	Valuer func() Valuer // 获取值对应的处理者，用于sql的scan、解析value等

	SQLValue func(val any) string // 转换为sql字符串值，用于insert等SQL语句的值转换

	// BindValue 把扫描得到的中间值归一为驱动可绑定的 Go 值（nil 代表无归一，驱动自行透传）。
	// 与 SQLValue 同构：nil → NULL 保留；“true”/“false” 文本与 Go bool 向数值列归一为 int64 1/0；
	// hex 文本向二进制列归一为 []byte；其他中间值默认透传，错误交给驱动自行拒绝，避免应用层“猜”非法数据。
	BindValue func(val any) (any, error)
}

// Copy 拷贝一个同类型的 ValueCodec，主要方便用于定制化修改 Valuer、SQLValue 或 BindValue
func (dt *ValueCodec) Copy() *ValueCodec {
	return &ValueCodec{
		Name:      dt.Name,
		Valuer:    dt.Valuer,
		SQLValue:  dt.SQLValue,
		BindValue: dt.BindValue,
	}
}

func (dt *ValueCodec) WithValuer(valuerFunc func() Valuer) *ValueCodec {
	dt.Valuer = valuerFunc
	return dt
}

func (dt *ValueCodec) WithSQLValue(sqlvalueFunc func(val any) string) *ValueCodec {
	dt.SQLValue = sqlvalueFunc
	return dt
}

func (dt *ValueCodec) WithBindValue(bindValueFunc func(val any) (any, error)) *ValueCodec {
	dt.BindValue = bindValueFunc
	return dt
}

// Bind 统一入口：nil 直接落 NULL；codec 未实现 BindValue 时透传（驱动默认接受中间值形态），
// 避免强制方言实现满员。与 SQLValue 对 nil 先判 NULL 的写法同构。
func (dt *ValueCodec) Bind(val any) (any, error) {
	if val == nil {
		return nil, nil
	}
	if dt.BindValue == nil {
		return val, nil
	}
	return dt.BindValue(val)
}

const NULL = "NULL"

// SQLValue 序列化函数族——扩展模型与命名约定（后续维护者请先读这段）：
//
// 本族每个函数是一种**独立的转义/字面量策略，按机制命名而非按方言命名**。方言不在本文件
// 分支选择策略，而是在自家 column.go 经 ValueCodec 槽位组合绑定：
//
//	DTStringMysql = dbi.DTString.Copy().WithSQLValue(dbi.SQLValueStringEscapeBackslash)
//
// 即 ValueCodec{Valuer, SQLValue} 函数槽就是方言的覆写点（组合优于继承：方言不子类化，
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

// ========== BindValue 函数族（与 SQLValue 同构的归一语义） ==========
//
// SQLValue 描述“拼入 SQL 文本时字面量长什么样”，BindValue 描述“驱动参数化接口里应绑成什么 Go 值”。
// 同一中间值经两条路径写入目标列必须落同结果——否则不同方言/不同插入策略会默默给出不同数据。
// 新方言只需覆写异常项（如自家驱动不认 int64），其他沿内置 DT* 的默认绑定。

// BindValueNumeric 数值/位/布尔目标列的绑定归一：
//   - Go bool → int64 1/0（ClickHouse DTBool、自定义中间值等）；
//   - "true"/"false" 字符串 → int64 1/0（PG Bool 经 DTString valuer 扫描后为字符串，而 MySQL
//     TINYINT 严格模式拒字符串、非严格默默存 0，与文本路径 SQLValueNumeric 输出裸 true/false
//     的隐式转语义分叉）；
//   - 其他值（包含数字文本、[]byte、int64 等）原样透传，交给驱动自行解析或报错，
//     应用层不尝试“猜”非法数据。
func BindValueNumeric(val any) (any, error) {
	switch v := val.(type) {
	case bool:
		if v {
			return int64(1), nil
		}
		return int64(0), nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true":
			return int64(1), nil
		case "false":
			return int64(0), nil
		}
	}
	return val, nil
}

// BindValueBytes 二进制列的绑定归一：
//   - []byte 原样；
//   - hex 字符串（ValuerBytes 扫描中介形态）→ []byte，与 SQLValueBytes 输出 X'...' 同构；
//     空串对应空字节集（旧实现 hex.DecodeString 直接能处理，不得以 IsHexString 误拒）；
//   - 非 hex 字符或其他类型报错，避免把不识别的形态当字符串写进 blob 造成默默失真。
func BindValueBytes(val any) (any, error) {
	switch v := val.(type) {
	case []byte:
		return v, nil
	case string:
		decoded, err := hex.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("invalid binary value %q: not a hex string: %w", v, err)
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("unsupported binary value type %T", val)
	}
}

// 内置数据类型全集：Valuer（扫描读取）、SQLValue（字面量写出）与 BindValue（参数化绑定）
// 三槽位的成对组合，各方言的类型清单（dialect/column.go）从这里取用并组合定制。
//
// SQLValue 与 BindValue 必须语义同构：同一中间值经两条路径写入目标列结果一致，
// 否则不同方言/不同插入策略会默默给出不同数据。新方言仅需覆写驱动不兼容项，
// 其他沿默认。无特殊归一需求的类型可不绑 BindValue（Bind 入口会透传）。
var (
	DTBit = &ValueCodec{
		Name:      "bit",
		Valuer:    ValuerBit,
		SQLValue:  SQLValueNumeric,
		BindValue: BindValueNumeric,
	}

	DTBool = &ValueCodec{
		Name:     "bool",
		Valuer:   ValuerBit,
		SQLValue: SQLValueBool,
		// BindValue 不绑定：目标列为 bool 时驱动接受 Go bool/标准布尔字面量，无需额外归一。
		// 向目标 int 列写 bool 的转换由目标 DTNumeric/DTInt* 的 BindValueNumeric 处理。
	}

	DTInt8 = &ValueCodec{
		Name:      "int8",
		Valuer:    ValuerInt16,
		SQLValue:  SQLValueNumeric,
		BindValue: BindValueNumeric,
	}

	DTInt16 = &ValueCodec{
		Name:      "int16",
		Valuer:    ValuerInt16,
		SQLValue:  SQLValueNumeric,
		BindValue: BindValueNumeric,
	}

	DTInt32 = &ValueCodec{
		Name:      "int32",
		Valuer:    ValuerInt32,
		SQLValue:  SQLValueNumeric,
		BindValue: BindValueNumeric,
	}

	DTInt64 = &ValueCodec{
		Name:      "int64",
		Valuer:    ValuerInt64,
		SQLValue:  SQLValueNumeric,
		BindValue: BindValueNumeric,
	}

	// 所有无符号类型，都使用int64存储
	DTUint64 = &ValueCodec{
		Name:      "uint64",
		Valuer:    ValuerUint64,
		SQLValue:  SQLValueNumeric,
		BindValue: BindValueNumeric,
	}

	// 使用string进行转换，避免长度过长导致精度丢失等
	DTNumeric = &ValueCodec{
		Name:      "numeric",
		Valuer:    ValuerString,
		SQLValue:  SQLValueNumeric,
		BindValue: BindValueNumeric,
	}

	DTDecimal = &ValueCodec{
		Name:      "decimal",
		Valuer:    ValuerString,
		SQLValue:  SQLValueNumeric,
		BindValue: BindValueNumeric,
	}

	DTString = &ValueCodec{
		Name:     "string",
		Valuer:   ValuerString,
		SQLValue: SQLValueString,
		// BindValue 不绑定：字符串列驱动接受 string/[]byte，无需归一。
	}

	DTDate = &ValueCodec{
		Name:     "date",
		Valuer:   ValuerDate,
		SQLValue: SQLValueDefault,
	}

	DTTime = &ValueCodec{
		Name:     "time",
		Valuer:   ValuerTime,
		SQLValue: SQLValueDefault,
	}

	DTDateTime = &ValueCodec{
		Name:     "datetime",
		Valuer:   ValuerDatetime,
		SQLValue: SQLValueDefault,
	}

	DTBytes = &ValueCodec{
		Name:      "bytes",
		Valuer:    ValuerBytes,
		SQLValue:  SQLValueBytes,
		BindValue: BindValueBytes,
	}
)
