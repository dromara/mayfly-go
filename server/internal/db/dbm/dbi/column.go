package dbi

import (
	"cmp"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"

	"github.com/may-fly/cast"
)

var (
	dbDataTypes = make(map[DbType]map[string]*DbDataType) // 列类型
)

// registerColumnDbDataTypes 注册数据库对应的数据类型
func registerColumnDbDataTypes(dbType DbType, cts ...*DbDataType) {
	dbDataTypes[dbType] = collx.ArrayToMap(cts, func(ct *DbDataType) string {
		return strings.ToLower(string(ct.Name))
	})
}

func GetDbDataType(dbType DbType, databaseColumnType string) *DbDataType {
	return cmp.Or(dbDataTypes[dbType][strings.ToLower(databaseColumnType)], DefaultDbDataType)
}

var DefaultDbDataType = NewDbDataType("string", DTString).WithCT(CTVarchar)

// 表的列信息
type Column struct {
	TableName     string  `json:"tableName"`     // 表名
	ColumnName    string  `json:"columnName"`    // 列名
	DataType      string  `json:"dataType"`      // 数据类型
	ColumnComment string  `json:"columnComment"` // 列备注
	IsPrimaryKey  bool    `json:"isPrimaryKey"`  // 是否为主键
	AutoIncrement bool    `json:"autoIncrement"` // 是否自增
	ColumnDefault string  `json:"columnDefault"` // 默认值
	Nullable      bool    `json:"nullable"`      // 是否可为null
	CharMaxLength int     `json:"charMaxLength"` // 字符最大长度
	NumPrecision  int     `json:"numPrecision"`  // 精度(总数字位数)
	NumScale      int     `json:"numScale"`      // 小数点位数
	Extra         collx.M `json:"extra"`         // 其他额外信息
}

// GetColumnType 获取完整的列类型，拼接数据类型与长度等。如varchar(2000)，decimal(20,2)
func (c *Column) GetColumnType() string {
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

func ClearNumScale(column *Column) {
	column.NumScale = 0
	column.CharMaxLength = 0
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

// SQLValueDefault 默认使用fmt转string
func SQLValueDefault(val any) string {
	if val == nil {
		return NULL
	}
	return fmt.Sprintf("'%v'", val)
}

// SQLValueNumeric 数字类型转string
func SQLValueNumeric(val any) string {
	if val == nil {
		return NULL
	}
	return fmt.Sprintf("%v", val)
}
func SQLValueBool(val any) string {
	if val == nil {
		return "false"
	}
	return fmt.Sprintf("%v", cast.ToBool(val))
}

func SQLValueString(val any) string {
	if val == nil {
		return NULL
	}

	strVal, ok := val.(string)
	if !ok {
		return fmt.Sprintf("%v", val)
	}

	return fmt.Sprintf("'%s'", strings.ReplaceAll(strings.ReplaceAll(strVal, "'", "''"), `\`, `\\`))
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
		SQLValue: SQLValueDefault,
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
	if valBytes == nil {
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
	if valBytes == nil {
		return nil
	}
	return valBytes[0]
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

// datetime

type datetimeValuer struct {
	*DefaultValuer[NullTime]
}

func (s *datetimeValuer) NewValuePtr() any {
	s.ValuePtr = &NullTime{
		Layout: time.DateTime,
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
		Layout: time.TimeOnly,
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
	default:
		return "", nil
	}
}
