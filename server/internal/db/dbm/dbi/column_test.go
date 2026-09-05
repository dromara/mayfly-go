package dbi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------- SQLValue 转换测试（数据导出/迁移的核心正确性） ----------

func TestSQLValueString(t *testing.T) {
	kases := []struct {
		name     string
		val      any
		expected string
	}{
		{"nil值", nil, "NULL"},
		{"普通字符串", "hello", "'hello'"},
		{"空字符串", "", "''"},
		{"单引号转义", "it's", "'it''s'"},
		{"连续单引号转义", "a''b", "'a''''b'"},
		{"反斜杠原样保留(标准SQL)", `a\b`, `'a\b'`},
		{"双引号原样保留", `he said "hi"`, `'he said "hi"'`},
		{"换行符原样保留", "line1\nline2", "'line1\nline2'"},
		{"非字符串-int", 123, "123"},
		{"非字符串-float", 1.5, "1.5"},
	}

	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.expected, SQLValueString(k.val))
		})
	}
}

func TestSQLValueStringEscapeBackslash(t *testing.T) {
	kases := []struct {
		name     string
		val      any
		expected string
	}{
		{"nil值", nil, "NULL"},
		{"普通字符串", "hello", "'hello'"},
		{"单引号转义", "it's", "'it''s'"},
		// mysql默认模式下反斜杠为转义字符，必须双写，否则 a\b 会被解释为 a<退格>
		{"反斜杠转义", `a\b`, `'a\\b'`},
		{"反斜杠与单引号同时转义", `a\b'c`, `'a\\b''c'`},
		{"双引号原样保留", `a"b`, `'a"b'`},
		{"非字符串-int", 123, "123"},
	}

	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.expected, SQLValueStringEscapeBackslash(k.val))
		})
	}
}

func TestSQLValuePreserveSpecialChars(t *testing.T) {
	// 保留特殊字符类型与标准SQL字符串转义行为一致
	assert.Equal(t, SQLValueString("a'b\"c\nd"), SQLValuePreserveSpecialChars("a'b\"c\nd"))
	assert.Equal(t, "'it''s'", SQLValuePreserveSpecialChars("it's"))
}

func TestSQLValueDefault(t *testing.T) {
	assert.Equal(t, "NULL", SQLValueDefault(nil))
	assert.Equal(t, "'123'", SQLValueDefault(123))
	assert.Equal(t, "'abc'", SQLValueDefault("abc"))
}

func TestSQLValueNumeric(t *testing.T) {
	assert.Equal(t, "NULL", SQLValueNumeric(nil))
	assert.Equal(t, "123", SQLValueNumeric(123))
	assert.Equal(t, "1.5", SQLValueNumeric(1.5))
	// 数字类型不加引号，防止隐式类型转换
	assert.Equal(t, "9999999999999999999", SQLValueNumeric("9999999999999999999"))
}

func TestSQLValueBool(t *testing.T) {
	assert.Equal(t, "false", SQLValueBool(nil))
	assert.Equal(t, "true", SQLValueBool(true))
	assert.Equal(t, "false", SQLValueBool(false))
	assert.Equal(t, "true", SQLValueBool(1))
}

// ---------- GetColumnType 拼接逻辑 ----------

func TestColumnGetColumnType(t *testing.T) {
	kases := []struct {
		name     string
		column   Column
		expected string
	}{
		{"已有完整列类型优先", Column{ColumnType: "varchar(2000)", DataType: "varchar", CharMaxLength: 100}, "varchar(2000)"},
		{"仅有字符最大长度", Column{DataType: "varchar", CharMaxLength: 255}, "varchar(255)"},
		{"精度与小数位", Column{DataType: "decimal", NumPrecision: 20, NumScale: 2}, "decimal(20,2)"},
		{"仅精度无小数位", Column{DataType: "int", NumPrecision: 10}, "int(10)"},
		{"均无则返回原始类型", Column{DataType: "text"}, "text"},
	}

	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.expected, k.column.GetColumnType())
		})
	}
}

// ---------- GetDbDataType 数据类型注册与查找 ----------

func TestGetDbDataType(t *testing.T) {
	// 未注册的数据库类型返回默认类型（string/DTString/CTVarchar）
	dt := GetDbDataType(DbType("not-exist-db"), "varchar")
	assert.Equal(t, DefaultDbDataType, dt)
	assert.Equal(t, CTVarchar, dt.CommonType)
	assert.Equal(t, DTString, dt.DataType)

	// 注册测试类型（小写作为key，查找不区分大小写）
	testDbType := DbType("test-dbtype-get")
	registerColumnDbDataTypes(testDbType,
		NewDbDataType("VARCHAR", DTString).WithCT(CTVarchar),
		NewDbDataType("INT8", DTInt64).WithCT(CTInt8),
	)

	assert.Equal(t, "VARCHAR", GetDbDataType(testDbType, "varchar").Name)
	assert.Equal(t, "VARCHAR", GetDbDataType(testDbType, "VARCHAR").Name)
	assert.Equal(t, "INT8", GetDbDataType(testDbType, "int8").Name)
	assert.Equal(t, CTInt8, GetDbDataType(testDbType, "int8").CommonType)
	// 未注册的类型名同样返回默认类型
	assert.Equal(t, DefaultDbDataType, GetDbDataType(testDbType, "unknown-type"))
}

// ---------- Valuer 行扫描值解析（迁移同步读取数据的正确性） ----------

func TestUint64Valuer(t *testing.T) {
	v := ValuerUint64()
	v.NewValuePtr() // 模拟真实scan链路：先创建值指针再取值

	// 空值场景：ptr指向空切片时应返回nil而不是0（区分NULL与0）
	assert.Equal(t, nil, v.Value())

	// 正常数值
	ptr := v.NewValuePtr().(*[]byte)
	*ptr = []byte("123")
	assert.Equal(t, uint64(123), v.Value())

	// 超过16位返回字符串防止前端精度丢失
	*ptr = []byte("12345678901234567890")
	assert.Equal(t, "12345678901234567890", v.Value())
}

func TestBitValuer(t *testing.T) {
	v := ValuerBit()
	v.NewValuePtr() // 模拟真实scan链路：先创建值指针再取值

	// 空切片返回nil（NULL），而非0
	assert.Equal(t, nil, v.Value())

	ptr := v.NewValuePtr().(*[]byte)
	*ptr = []byte{1}
	assert.Equal(t, byte(1), v.Value())

	*ptr = []byte{0}
	assert.Equal(t, byte(0), v.Value())
}

func TestStringValuer(t *testing.T) {
	sv := ValuerString()
	sv.NewValuePtr() // 模拟真实scan链路：先创建值指针再取值
	// Valid=false 返回nil（NULL）
	assert.Equal(t, nil, sv.Value())

	// 正常字符串
	sv.(*stringValuer).ValuePtr.String = "abc"
	sv.(*stringValuer).ValuePtr.Valid = true
	assert.Equal(t, "abc", sv.Value())
}

func TestInt64Valuer(t *testing.T) {
	v := ValuerInt64()

	v.NewValuePtr()
	ptr := v.(*int64Valuer)
	ptr.ValuePtr.Valid = true
	ptr.ValuePtr.Int64 = 9999999999999999 // 16位，返回int64
	assert.Equal(t, int64(9999999999999999), v.Value())

	ptr.ValuePtr.Int64 = 99999999999999999 // 超过16位，返回字符串防精度丢失
	assert.Equal(t, "99999999999999999", v.Value())

	ptr.ValuePtr.Valid = false
	assert.Equal(t, nil, v.Value())
}
