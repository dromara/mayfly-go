package dbi

import (
	"cmp"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
)

var (
	dbDataTypes      = make(map[DbType]map[string]*DbDataType) // 列类型
	dbDataTypesMutex sync.RWMutex                              // 读写锁
)

// registerColumnDbDataTypes 注册数据库对应的数据类型
func registerColumnDbDataTypes(dbType DbType, cts ...*DbDataType) {
	dbDataTypesMutex.Lock()
	defer dbDataTypesMutex.Unlock()

	dbDataTypes[dbType] = collx.ArrayToMap(cts, func(ct *DbDataType) string {
		return strings.ToLower(string(ct.Name))
	})
}

func GetDbDataType(dbType DbType, databaseColumnType string) *DbDataType {
	dbDataTypesMutex.RLock()
	defer dbDataTypesMutex.RUnlock()

	return cmp.Or(dbDataTypes[dbType][strings.ToLower(databaseColumnType)], DefaultDbDataType)
}

var DefaultDbDataType = NewDbDataType("string", DTString).WithCT(CTVarchar)

// ColumnExtraOnUpdate Column.Extra的key：源库元数据里的「自动更新」子句原文（如MySQL的
// on update CURRENT_TIMESTAMP(3)）。属于方言专有语法，故以扩展信息而非独立字段承载，
// 仅目标方言自身支持时（MySQL生成器）才写入DDL
const ColumnExtraOnUpdate = "onUpdate"

// 生成列的派生信息（存于Column.Extra，方言专有语法且长度不定，不占用独立字段）：
// 表达式文本属于源方言SQL，跨方言无法保证其语法与函数存在，故必须连同源库类型一起记录
const (
	// ColumnExtraGenExpr 生成列的派生表达式原文（如MySQL的 (`a` + `b`)、pg的 (c1 * 2)）
	ColumnExtraGenExpr = "genExpr"
	// ColumnExtraGenKind 生成列的物化方式：stored（结果物化存储，如MySQL STORED、pg STORED）
	// 或virtual（查询时计算）
	ColumnExtraGenKind = "genKind"
	// ColumnExtraGenDbType 表达式所属的源数据库类型，仅目标与源同方言时才可原样重建
	ColumnExtraGenDbType = "genDbType"
)

// 生成列物化方式的取值（与各方言元数据的存储语义对齐）
const (
	GenerationStored  = "stored"
	GenerationVirtual = "virtual"
)

// MarkGeneratedColumn 记录生成列的派生表达式与物化方式，供同方言迁移重建DDL以及插入时剔除该列使用。
// 仅当源库能交出完整的派生表达式文本时才记录：无表达式则无法重建，必须退回「目标建普通列 + 插入源值」的语义，
// 否则会出现目标建成普通列却又不插入该列的静默NULL失真（比直接报错更隐蔽）
func MarkGeneratedColumn(column *Column, srcDbType, expr, kind string) {
	if column == nil || expr == "" {
		return
	}
	column.IsGenerated = true
	if column.Extra == nil {
		column.Extra = collx.M{}
	}
	column.Extra[ColumnExtraGenExpr] = expr
	column.Extra[ColumnExtraGenKind] = kind
	column.Extra[ColumnExtraGenDbType] = srcDbType
}

// PreservableGeneratedColumn 生成列能否在目标方言下原样重建（而非退化为普通列）：
// 必须同时满足已取到派生表达式、且目标与表达式所属源库为同一方言。
//
// DDL生成与INSERT剔除必须共用本判定，否则会出现“目标建成了普通列却又不插入该列”
// 导致生成列值静默变NULL（比直接报错更隐蔽）的数据失真
func PreservableGeneratedColumn(column Column, targetDbType DbType) bool {
	if !column.IsGenerated {
		return false
	}
	expr, _ := column.Extra[ColumnExtraGenExpr].(string)
	if strings.TrimSpace(expr) == "" {
		return false
	}
	srcDbType, _ := column.Extra[ColumnExtraGenDbType].(string)
	return srcDbType == string(targetDbType)
}

// GeneratedColumnExpr 取生成列的派生表达式原文，无则返回空字符串
func GeneratedColumnExpr(column Column) string {
	expr, _ := column.Extra[ColumnExtraGenExpr].(string)
	return strings.TrimSpace(expr)
}

// GeneratedColumnStored 生成列是否为物化存储（stored）：仅物化存储的列才可安全重建，
// 因为部分方言语义上不支持虚拟生成列（如pg 12/13只有STORED，VIRTUAL自pg 18才存在）
func GeneratedColumnStored(column Column) bool {
	kind, _ := column.Extra[ColumnExtraGenKind].(string)
	return kind == GenerationStored
}

// 表的列信息
type Column struct {
	TableName     string `json:"tableName"`     // 表名
	ColumnName    string `json:"columnName"`    // 列名
	ColumnType    string `json:"columnType"`    // 完整列类型，带有数据类型以及长度、精度等。如varchar(2000)，decimal(20,2)
	DataType      string `json:"dataType"`      // 数据类型
	ColumnComment string `json:"columnComment"` // 列备注
	IsPrimaryKey  bool   `json:"isPrimaryKey"`  // 是否为主键
	AutoIncrement bool   `json:"autoIncrement"` // 是否自增
	ColumnDefault string `json:"columnDefault"` // 默认值
	Nullable      bool   `json:"nullable"`      // 是否可为null
	// IsExprDefault 源库元数据表明 ColumnDefault 是「表达式默认值」（函数调用/运算式）而非字面量内容。
	// 各库表达式语法与函数名互不相通（pg的gen_random_uuid()、mysql的concat(_latin1\'x\')），跨库无法还原，
	// 生成DDL时必须省略该默认值；若按字面量写出（DEFAULT 'gen_random_uuid()'），uuid/jsonb列建表即报错，
	// 字符串列则静默把函数名当成默认值内容（数据污染）。由各方言metadata读取时按自身呈现约定标记
	IsExprDefault bool `json:"isExprDefault"`
	// IsGenerated 是否为生成列（MySQL的VIRTUAL/STORED GENERATED、SQL Server的计算列、Oracle的虚拟列等）：
	// 其值由表达式派生，不可显式INSERT（MySQL报Error 3105、pg报cannot insert a non-DEFAULT value），
	// 数据迁移/导出导入必须从插入列集中剔除；COLUMN_DEFAULT存的是派生表达式而非默认值，也不得当默认值写出
	IsGenerated   bool    `json:"isGenerated"`
	CharMaxLength int     `json:"charMaxLength"` // 字符最大长度
	NumPrecision  int     `json:"numPrecision"`  // 精度(总数字位数)
	NumScale      int     `json:"numScale"`      // 小数点位数
	Extra         collx.M `json:"extra"`         // 其他额外信息
}

// SplitColumnTypeBase 拆分列类型书写形态为其基础类型名与第一个括号参数的数值：
// "datetime(3)"→("datetime",3,true)，"decimal(20,6)"→("decimal",20,true)，"varchar"→("varchar",0,false)。
// 用于方言生成器按目标列最终书写的小数秒精度归一默认值参数（如MySQL要求CURRENT_TIMESTAMP(fsp)与列fsp一致）
func SplitColumnTypeBase(columnType string) (string, int, bool) {
	lower := strings.ToLower(strings.TrimSpace(columnType))
	open := strings.IndexByte(lower, '(')
	if open < 0 || !strings.HasSuffix(lower, ")") {
		return lower, 0, false
	}
	base := strings.TrimSpace(lower[:open])
	inner := strings.TrimSpace(lower[open+1 : len(lower)-1])
	// 取第一个逗号前的数值（decimal(20,6)的精度部分），非纯数字则视为无有效参数
	if idx := strings.IndexByte(inner, ','); idx >= 0 {
		inner = inner[:idx]
	}
	num := 0
	if inner == "" {
		return base, 0, false
	}
	for i := 0; i < len(inner); i++ {
		if inner[i] < '0' || inner[i] > '9' {
			return base, 0, false
		}
		num = num*10 + int(inner[i]-'0')
	}
	return base, num, true
}

// GetColumnType 获取完整的列类型，拼接数据类型与长度等。如varchar(2000)，decimal(20,2)
func (c *Column) GetColumnType() string {
	if c.ColumnType != "" {
		return c.ColumnType
	}

	if c.CharMaxLength > 0 {
		return fmt.Sprintf("%s(%d)", c.DataType, c.CharMaxLength)
	}
	if c.NumPrecision > 0 {
		if c.NumScale > 0 {
			return fmt.Sprintf("%s(%d,%d)", c.DataType, c.NumPrecision, c.NumScale)
		} else {
			return fmt.Sprintf("%s(%d)", c.DataType, c.NumPrecision)
		}
	}

	return c.DataType
}

// 数据库对应的数据类型
type DbDataType struct {
	Name string //  类型名

	DataType *DataType // 数据类型

	fixColumnFunc func(column *Column) // 修复字段长度、精度等, 如mysql text会返回长度，需要将其置为0等

	/** 以下为异构数据迁移同步使用，可不赋值，无值则不支持迁移同步 */

	CommonType CommonDbDataType // 对应的公共类型
}

// WithFixColumn 修复列信息函数，用于修复字段长度、精度等
func (ct *DbDataType) WithFixColumn(fixColumnFunc func(column *Column)) *DbDataType {
	ct.fixColumnFunc = fixColumnFunc
	return ct
}

// WithCT 对应的公共类型，主要用于异构数据库迁移同步时进行类型转换使用
func (ct *DbDataType) WithCT(cct CommonDbDataType) *DbDataType {
	ct.CommonType = cct
	return ct
}

// FixColumn 使用修复列信息函数进行列信息修复
func (ct *DbDataType) FixColumn(column *Column) {
	if ct.fixColumnFunc != nil {
		ct.fixColumnFunc(column)
	}
}

func NewDbDataType(name string, dataType *DataType) *DbDataType {
	return &DbDataType{
		Name:     name,
		DataType: dataType,
	}
}

func ClearCharMaxLength(column *Column) {
	column.CharMaxLength = 0
	column.NumPrecision = 0
}

// ClearCharLength 仅清空字符长度：时间类列的元数据不含字符长度但含小数秒精度（存于NumPrecision），
// 使用ClearCharMaxLength会一并抹掉精度，使异构迁移后小数秒静默丢失
func ClearCharLength(column *Column) {
	column.CharMaxLength = 0
}

func ClearNumScale(column *Column) {
	column.NumScale = 0
	column.CharMaxLength = 0
}

func ClearNumPrecision(column *Column) {
	column.NumScale = 0
	column.NumPrecision = 0
	column.CharMaxLength = 0
}

// FillUnboundedDecimal 源列为无精度约束的精确数值（如pg的numeric、oracle的NUMBER、sqlite声明的numeric）时，
// 按目标库上限补齐精度与小数位。
// MySQL的decimal省略精度等价decimal(10,0)、SQL Server的numeric默认(18,0)，若沿用无精度的源列，
// 目标表建出来后所有小数都会被静默截断（结构迁移后数据失真且无报错），故必须补齐为尽可能宽的精度
func FillUnboundedDecimal(column *Column, maxPrecision, maxScale int) {
	if column == nil || column.NumPrecision > 0 {
		return
	}
	column.NumPrecision = maxPrecision
	column.NumScale = maxScale
}

// ClampDecimalPrecision 将精度与小数位收敛到目标库支持范围内，并保证小数位不超过精度。
// 源库精度上限高于目标库时（如mysql decimal(65,30) -> oracle NUMBER(38,x)），不收敛会直接生成非法DDL
func ClampDecimalPrecision(column *Column, maxPrecision, maxScale int) {
	if column == nil || column.NumPrecision <= 0 {
		return
	}
	if column.NumScale < 0 {
		column.NumScale = 0
	}
	if column.NumScale > maxScale {
		column.NumScale = maxScale
	}
	if column.NumPrecision > maxPrecision {
		column.NumPrecision = maxPrecision
	}
	if column.NumScale > column.NumPrecision {
		column.NumScale = column.NumPrecision
	}
}

// NormalizeTimeFsp 归一化时间类列的小数秒精度（fsp），用于目标库支持且需要显式声明fsp的情况：
//   - NumScale对时间类型无语义，必须清零，否则会被GetColumnType拼成 datetime(6,2) 这类非法DDL
//   - 源库未提供fsp时按目标库最大fsp补齐：MySQL的datetime省略fsp即datetime(0)，源值的小数秒会在迁移后静默丢失
//   - 源fsp大于目标库上限时收敛到上限（如mssql datetime2(7) -> mysql datetime(6)）
func NormalizeTimeFsp(column *Column, maxFsp int) {
	if column == nil {
		return
	}
	column.NumScale = 0
	if column.NumPrecision <= 0 || column.NumPrecision > maxFsp {
		column.NumPrecision = maxFsp
	}
}

// ClampTimeFsp 仅收敛超出目标库上限的小数秒精度：目标库不声明fsp时即为自身最大精度（如pg的timestamp）时使用，
// 此时无需也不应主动补齐精度
func ClampTimeFsp(column *Column, maxFsp int) {
	if column == nil {
		return
	}
	column.NumScale = 0
	if column.NumPrecision > maxFsp {
		column.NumPrecision = maxFsp
	}
}

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

// Valuer 获取值对应的处理者，用于sql row scan、解析value等
type Valuer interface {

	// NewValuePtr 新建值对应的指针，用于sql的row scan
	NewValuePtr() any

	// Value 获取对应的值（人类可阅读的值），不可原样返回ValuePtr指针类型，需取出具体的值
	Value() any
}

type DefaultValuer[T any] struct {
	ValuePtr *T
}

func (s *DefaultValuer[T]) NewValuePtr() any {
	var t T
	s.ValuePtr = &t
	return s.ValuePtr
}

// Valuer工厂函数

func ValuerString() Valuer {
	return &stringValuer{
		DefaultValuer: new(DefaultValuer[sql.NullString]),
	}
}

func ValuerInt64() Valuer {
	return &int64Valuer{
		DefaultValuer: new(DefaultValuer[sql.NullInt64]),
	}
}

func ValuerUint64() Valuer {
	return &uint64Valuer{
		DefaultValuer: new(DefaultValuer[[]byte]),
	}
}

func ValuerInt32() Valuer {
	return &int32Valuer{
		DefaultValuer: new(DefaultValuer[sql.NullInt32]),
	}
}

func ValuerInt16() Valuer {
	return &int16Valuer{
		DefaultValuer: new(DefaultValuer[sql.NullInt16]),
	}
}

func ValuerByte() Valuer {
	return &byteValuer{
		DefaultValuer: new(DefaultValuer[sql.NullByte]),
	}
}

func ValuerBit() Valuer {
	return &bitValuer{
		DefaultValuer: new(DefaultValuer[[]byte]),
	}
}

func ValuerFloat64() Valuer {
	return &float64Valuer{
		DefaultValuer: new(DefaultValuer[sql.NullFloat64]),
	}
}

func ValuerDatetime() Valuer {
	return &datetimeValuer{
		DefaultValuer: new(DefaultValuer[NullTime]),
	}
}

func ValuerDate() Valuer {
	return &dateValuer{
		DefaultValuer: new(DefaultValuer[NullTime]),
	}
}

func ValuerTime() Valuer {
	return &timeValuer{
		DefaultValuer: new(DefaultValuer[NullTime]),
	}
}

func ValuerBytes() Valuer {
	return &bytesValuer{
		DefaultValuer: new(DefaultValuer[sql.RawBytes]),
	}
}

// 默认 valuer

// string

type stringValuer struct {
	*DefaultValuer[sql.NullString]
}

func (s *stringValuer) Value() any {
	if s.ValuePtr.Valid {
		return s.ValuePtr.String
	}
	return nil
}

// uint64

type uint64Valuer struct {
	*DefaultValuer[[]byte]
}

func (s *uint64Valuer) Value() any {
	valBytes := *s.ValuePtr
	if len(valBytes) == 0 {
		return nil
	}
	val := string(valBytes)
	// 前端超过16位会丢失精度
	if len(val) > 16 {
		return val
	}
	return cast.ToUint64(val)
}

//  int64

type int64Valuer struct {
	*DefaultValuer[sql.NullInt64]
}

func (s *int64Valuer) Value() any {
	if s.ValuePtr.Valid {
		val := s.ValuePtr.Int64
		// 前端超过16位会丢失精度
		if val > 9999999999999999 {
			return fmt.Sprintf("%d", val)
		}
		return val
	}
	return nil
}

// int32

type int32Valuer struct {
	*DefaultValuer[sql.NullInt32]
}

func (s *int32Valuer) Value() any {
	if s.ValuePtr.Valid {
		return s.ValuePtr.Int32
	}
	return nil
}

// int16

type int16Valuer struct {
	*DefaultValuer[sql.NullInt16]
}

func (s *int16Valuer) Value() any {
	if s.ValuePtr.Valid {
		return s.ValuePtr.Int16
	}
	return nil
}

// byte（uint8）

type byteValuer struct {
	*DefaultValuer[sql.NullByte]
}

func (s *byteValuer) Value() any {
	if s.ValuePtr.Valid {
		return s.ValuePtr.Byte
	}
	return nil
}

// bit

type bitValuer struct {
	*DefaultValuer[[]byte]
}

func (s *bitValuer) Value() any {
	valBytes := *s.ValuePtr
	if len(valBytes) == 0 {
		return nil
	}
	// driver对BIT(N)返回大端字节（BIT(1-8)为1字节，最大BIT(64)为8字节），
	// 仅取首字节会导致BIT(9-64)的高位丢失（静默截断），需按大端合成整数值
	if len(valBytes) > 8 {
		// 超过BIT(64)的异常数据，丢弃高位仅保留低8字节，避免panic
		valBytes = valBytes[len(valBytes)-8:]
	}
	buf := make([]byte, 8)
	copy(buf[8-len(valBytes):], valBytes)
	uval := binary.BigEndian.Uint64(buf)
	if uval <= math.MaxInt64 {
		return int64(uval)
	}
	return uval
}

// float64

type float64Valuer struct {
	*DefaultValuer[sql.NullFloat64]
}

func (s *float64Valuer) Value() any {
	if s.ValuePtr.Valid {
		return s.ValuePtr.Float64
	}
	return nil
}

// bytes

type bytesValuer struct {
	*DefaultValuer[sql.RawBytes]
}

func (s *bytesValuer) Value() any {
	val := s.ValuePtr
	if *val == nil {
		return nil
	}
	return hex.EncodeToString(*val)
}

// datetimeLayout/timeLayout 保留至微秒（.999999 会去除末尾多余的0）：
// datetime(3)/datetime(6)、timestamp(3)/(6)等列在驱动返回time.Time时（mysql parseTime=true、pg lib/pq、
// sqlite modernc驱动按decltype解析），若仅用time.DateTime格式化会静默丢弃小数秒，
// 导致导出/迁移后的时间值与源库不一致（基础设施场景下不可接受）
const (
	datetimeLayout = time.DateTime + ".999999"
	timeLayout     = time.TimeOnly + ".999999"
)

// datetime

type datetimeValuer struct {
	*DefaultValuer[NullTime]
}

func (s *datetimeValuer) NewValuePtr() any {
	s.ValuePtr = &NullTime{
		Layout: datetimeLayout,
	}
	return s.ValuePtr
}

func (s *datetimeValuer) Value() any {
	if s.ValuePtr.Valid {
		return s.ValuePtr.Time
	}
	return nil
}

// date

type dateValuer struct {
	*DefaultValuer[NullTime]
}

func (s *dateValuer) NewValuePtr() any {
	s.ValuePtr = &NullTime{
		Layout: time.DateOnly,
	}
	return s.ValuePtr
}

func (s *dateValuer) Value() any {
	if s.ValuePtr.Valid {
		return s.ValuePtr.Time
	}
	return nil
}

// time

type timeValuer struct {
	*DefaultValuer[NullTime]
}

func (s *timeValuer) NewValuePtr() any {
	s.ValuePtr = &NullTime{
		Layout: timeLayout,
	}
	return s.ValuePtr
}

func (s *timeValuer) Value() any {
	if s.ValuePtr.Valid {
		return s.ValuePtr.Time
	}
	return nil
}

// NullTime represents a time that may be null.
// NullTime implements the [Scanner] interface so
// it can be used as a scan destination, similar to [NullString].
type NullTime struct {
	Time   string
	Valid  bool // Valid is true if Time is not NULL
	Layout string
}

var (
	_ driver.Valuer = NullTime{}
)

// Scan implements the [Scanner] interface.
func (n *NullTime) Scan(value any) error {
	if value == nil {
		n.Time, n.Valid = "", false
		return nil
	}

	n.Valid = true
	time, err := convertTime(value, n.Layout)
	if err != nil {
		return err
	}
	n.Time = time
	return nil
}

// Value implements the driver Valuer interface.
func (n NullTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

func convertTime(src interface{}, layout string) (string, error) {
	switch s := src.(type) {
	case string:
		return s, nil
	case []uint8:
		return string(s), nil
	case time.Time:
		return s.Format(layout), nil
	case *time.Time:
		if s == nil {
			return "", nil
		}
		return s.Format(layout), nil
	default:
		// 未知驱动类型不可静默返回空串（会将真实时间值写成空值/NULL，属数据损坏），必须显式报错
		return "", fmt.Errorf("unsupported time value type: %T", src)
	}
}
