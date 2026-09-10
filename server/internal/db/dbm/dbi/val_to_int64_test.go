package dbi

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 驱动返值形态归一化的无DB单测。
//
// 该组函数是导入/迁移链路的「取值底座」：分片边界（MIN/MAX主键）、行数校验（COUNT）、
// 布尔与浮点比对都经此归一。形态漏一个分支不会报错，只会让上层走兜底路径——
// 表现为并行分片静默退化为单任务、或校验被跳过，属最难察觉的一类缺陷，故按形态矩阵逐个钉死。
//
// 期望值来源不是推测，而是本机同一张 id SMALLINT PRIMARY KEY 表的聚合查询实测：
//   - mysql 8.0：MIN/MAX(smallint)→int16，COUNT(*)→int64，SUM(smallint)→string
//   - postgres 16：MIN/MAX(smallint)→int16，COUNT(*)→int64，SUM→int64
//   - SQL Server 2022：TINYINT/SMALLINT→int16，INT→int32，BIGINT→int64，COUNT(*)→int32，@@SPID→int16
//   - sqlite：数值聚合一律返回string（弱类型文本 affinity）

// TestValToInt64DriverForms 各Go形态的转换结果矩阵（含各方言实测的窄整型）
func TestValToInt64DriverForms(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int64
		ok   bool
	}{
		{"nil", nil, 0, false},
		{"int64(bigint列)", int64(9223372036854775807), math.MaxInt64, true},
		{"int", int(42), 42, true},
		{"int8", int8(-128), -128, true},
		{"int16(mysql/pg/mssql 的 smallint MIN/MAX)", int16(32767), 32767, true},
		{"int16负值", int16(-32768), -32768, true},
		{"int32(mssql int列)", int32(-2147483648), -2147483648, true},
		{"uint", uint(7), 7, true},
		{"uint8", uint8(255), 255, true},
		{"uint16", uint16(65535), 65535, true},
		{"uint32", uint32(4294967295), 4294967295, true},
		{"uint64", uint64(12345678), 12345678, true},
		{"float32", float32(1.9), 1, true},
		{"float64(如oracle number)", float64(1.0), 1, true},
		{"[]byte(mysql数值文本)", []byte("100"), 100, true},
		{"string", "200", 200, true},
		{"bool非数值", true, 0, false},
		{"time非数值", time.Now(), 0, false},
		{"结构体非数值", struct{ a int }{1}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ValToInt64(tt.val)
			assert.Equal(t, tt.ok, ok, "值=%#v", tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValToInt64CoversAllIntKinds 遍历Go全部整型类型，防新增窄类型再次漏分支。
//
// 与手写枚举互补：枚举靠人记得住，此用例保证「任何Go整型形态都不应返回false」
func TestValToInt64CoversAllIntKinds(t *testing.T) {
	ptrs := []any{new(int), new(int8), new(int16), new(int32), new(int64),
		new(uint), new(uint8), new(uint16), new(uint32), new(uint64), new(uintptr)}
	for _, ptr := range ptrs {
		v := reflect.ValueOf(ptr).Elem()
		kind := v.Kind()
		switch {
		case kind >= reflect.Int && kind <= reflect.Int64:
			v.SetInt(5)
		default:
			v.SetUint(5)
		}
		t.Run(kind.String(), func(t *testing.T) {
			got, ok := ValToInt64(v.Interface())
			if kind == reflect.Uintptr {
				assert.False(t, ok, "uintptr非驱动返值形态，不应静默接受")
				return
			}
			assert.True(t, ok, "整型Kind [%s] 未被识别，上层会静默走兜底路径", kind)
			assert.Equal(t, int64(5), got)
		})
	}
}

// TestShardablePkTypesAllConvertibleByValToInt64 分片主键白名单与取值归一化必须闭环。
//
// 缺陷现场：DetectIntPrimaryKey把tinyint/smallint判为可分片，而mssql驱动对这两类列的
// MIN/MAX返回int16，ValToInt64当时不识别 → planTableShards返回nil → 大表并行迁移静默退化为单任务
func TestShardablePkTypesAllConvertibleByValToInt64(t *testing.T) {
	cols := func(dataType string) []Column {
		return []Column{{TableName: "t", ColumnName: "id", DataType: dataType, IsPrimaryKey: true}}
	}
	// 白名单内的每个整型SQL类型，其可能对应的Go窄整型形态都必须可归一
	narrowForms := []any{int8(3), int16(3), int32(3), int64(3), uint8(3), uint16(3), uint32(3), []byte("3"), "3"}
	for _, dt := range []string{"tinyint", "smallint", "mediumint", "int", "integer", "bigint", "int2", "int4", "int8"} {
		assert.Equal(t, "id", DetectIntPrimaryKey(cols(dt)), "整型主键 [%s] 应判为可分片", dt)
		for _, form := range narrowForms {
			got, ok := ValToInt64(form)
			assert.True(t, ok, "主键类型[%s]的MIN/MAX返值形态 %T 无法归一，分片会静默退化", dt, form)
			assert.Equal(t, int64(3), got)
		}
	}
}

// TestParseStrToInt64 文本形态：定宽空格、浮点式文本、越界与非数值
func TestParseStrToInt64(t *testing.T) {
	tests := []struct {
		text string
		want int64
		ok   bool
	}{
		{"42", 42, true},
		{" 42 ", 42, true},
		{"\n42\r", 42, true},
		{"-7", -7, true},
		{"1.0", 1, true},
		{" 1.0 ", 1, true},
		{"1e3", 1000, true},
		{"9223372036854775807", math.MaxInt64, true},
		{"-9223372036854775808", math.MinInt64, true},
		// 越界：float64→int64为未定义行为，必须显式拒绝而非返回随机值
		{"9223372036854775808", 0, false},
		{"-9300000000000000000", 0, false},
		{"1e30", 0, false},
		{"1e999", 0, false},
		{"NaN", 0, false},
		{"Inf", 0, false},
		{"", 0, false},
		{"abc", 0, false},
		{"1_000", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got, ok := ParseStrToInt64(tt.text)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValToFloat64DriverForms 浮点归一矩阵（REAL/DOUBLE/FLOAT列与decimal文本）
func TestValToFloat64DriverForms(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want float64
		ok   bool
	}{
		{"nil", nil, 0, false},
		{"float64", float64(1.5), 1.5, true},
		{"float32", float32(1.5), 1.5, true},
		{"int8", int8(-3), -3, true},
		{"int16", int16(300), 300, true},
		{"int32", int32(-66000), -66000, true},
		{"int64", int64(7), 7, true},
		{"uint8", uint8(200), 200, true},
		{"uint16", uint16(65535), 65535, true},
		{"uint32", uint32(4000000000), 4e9, true},
		{"uint64", uint64(9), 9, true},
		{"[]byte", []byte("1.25"), 1.25, true},
		{"string", " -0.5 ", -0.5, true},
		{"非数值string", "abc", 0, false},
		{"bool", true, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ValToFloat64(tt.val)
			assert.Equal(t, tt.ok, ok)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

// TestValToBoolDriverForms 布尔归一矩阵：各方言bool列物理类型差异大（tinyint/bit/boolean/'t'文本）
func TestValToBoolDriverForms(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
		ok   bool
	}{
		{"nil", nil, false, false},
		{"bool_true", true, true, true},
		{"bool_false", false, false, true},
		{"int64_0(mysql tinyint(1))", int64(0), false, true},
		{"int64_1", int64(1), true, true},
		{"int8_1", int8(1), true, true},
		{"int16_0", int16(0), false, true},
		{"int32_2", int32(2), true, true},
		{"uint8_1", uint8(1), true, true},
		{"float64_0", float64(0), false, true},
		{"string_t(postgres)", "t", true, true},
		{"string_f(postgres)", "f", false, true},
		{"string_TRUE", " TRUE ", true, true},
		{"string_1", "1", true, true},
		{"string_0", []byte("0"), false, true},
		{"string_非法", "maybe", false, false},
		{"struct", struct{}{}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ValToBool(tt.val)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseStrToFloat64 浮点文本解析的空格与非法值口径
func TestParseStrToFloat64(t *testing.T) {
	for _, tt := range []struct {
		text string
		want float64
		ok   bool
	}{
		{"1.5", 1.5, true},
		{" 1.5\n", 1.5, true},
		{"1e-10", 1e-10, true},
		{"", 0, false},
		{"1.2.3", 0, false},
		{"abc", 0, false},
	} {
		t.Run(tt.text, func(t *testing.T) {
			got, ok := ParseStrToFloat64(tt.text)
			assert.Equal(t, tt.ok, ok)
			assert.InDelta(t, tt.want, got, 1e-12)
		})
	}
}
