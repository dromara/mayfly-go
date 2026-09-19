package dbi

import (
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/spf13/cast"
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

func convertTime(src any, layout string) (string, error) {
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
