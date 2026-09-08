package dbi

import (
	"math"
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
		{"非字符串-int必须仍为带引号字面量", 123, "'123'"},
		{"非字符串-float必须仍为带引号字面量", 1.5, "'1.5'"},
		// []byte必须取字节内容，旧实现%v会打印为“[97 98 99]”写入目标库造成数据损坏
		{"字节数组取内容", []byte("abc"), "'abc'"},
		{"字节数组含单引号", []byte("a'b"), "'a''b'"},
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
		{"非字符串-int必须仍为带引号字面量", 123, "'123'"},
		{"字节数组取内容", []byte("a\\b"), `'a\\b'`},
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
	// 日期/时间类型的库内文本同样可能含单引号（如异构脏数据），必须转义而非裸拼
	assert.Equal(t, "'it''s'", SQLValueDefault("it's"))
	assert.Equal(t, `'2020-01-01 00:00:00'`, SQLValueDefault("2020-01-01 00:00:00"))
	// 含分号的文本不得被切割为独立语句
	assert.Equal(t, `'a; DROP TABLE t; --'`, SQLValueDefault("a; DROP TABLE t; --"))
}

func TestIsNumericLiteral(t *testing.T) {
	for _, valid := range []string{"123", "-5", "+7", "1.5", "0.25", "1e10", "-2.5E-3", "0"} {
		assert.True(t, IsNumericLiteral(valid), "should be numeric literal: %s", valid)
	}
	for _, invalid := range []string{"", " ", "abc", "1; DROP TABLE t", "0x1f", "1_000", "Inf", "NaN", ".", "1.", "e5", "1e", "--", "1 2", "+", "1.2.3"} {
		assert.False(t, IsNumericLiteral(invalid), "should not be numeric literal: %s", invalid)
	}
}

func TestIsPlainSqlLiteral(t *testing.T) {
	for _, valid := range []string{"123", "-1.5", "0x1f", "0XABCDEF", "b'01'", "B'1'", "CURRENT_TIMESTAMP", "null", "TRUE"} {
		assert.True(t, IsPlainSqlLiteral(valid), "should be plain literal: %s", valid)
	}
	for _, invalid := range []string{"", "abc", "it's", "'quoted'", "0xzz", "b'02'", "1; DROP TABLE t", "b'"} {
		assert.False(t, IsPlainSqlLiteral(invalid), "should not be plain literal: %s", invalid)
	}
}

func TestUnwrapSqlLiteral(t *testing.T) {
	kases := []struct {
		name     string
		val      string
		expected string
	}{
		// 无引号包裹（MySQL 8.0元数据直接返回原始值）原样返回
		{"未包裹原样返回", "abc", "abc"},
		{"未包裹含单引号不失真", "end'", "end'"},
		{"未包裹首尾均含引号", "a'b", "a'b"},
		// 带引号包裹（5.7/MariaDB/sqlite dflt_value）剥一层并还原双写
		{"剥外层引号", "'abc'", "abc"},
		{"剥外层并还原双写", "'it''s'", "it's"},
		{"空字面量", "''", ""},
		// 仅剥最外层一对：旧实现按字符集剥除所有引号会使以下内容静默失真
		{"仅剥一层", "'''quoted'''", "'quoted'"},
		{"单个引号字符", "''''", "'"},
		{"内层双写不多剥", "'a''''b'", "a''b"},
	}
	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.expected, UnwrapSqlLiteral(k.val))
		})
	}
}

func TestSQLValueNumeric(t *testing.T) {
	assert.Equal(t, "NULL", SQLValueNumeric(nil))
	assert.Equal(t, "123", SQLValueNumeric(123))
	assert.Equal(t, "1.5", SQLValueNumeric(1.5))
	// 数字类型不加引号，防止隐式类型转换
	assert.Equal(t, "9999999999999999999", SQLValueNumeric("9999999999999999999"))
	assert.Equal(t, "-0.000125", SQLValueNumeric("-0.000125"))
	// 弱类型库/脏数据可能使数字列存非法文本：退化为转义字面量，不得裸拼出可执行语句
	assert.Equal(t, "'1; DROP TABLE t; --'", SQLValueNumeric("1; DROP TABLE t; --"))
	assert.Equal(t, "'NaN'", SQLValueNumeric("NaN"))
	assert.Equal(t, "'+Inf'", SQLValueNumeric(math.Inf(1)))
	assert.Equal(t, "'abc'", SQLValueNumeric("abc"))
}

func TestSQLValueBool(t *testing.T) {
	// 与其它SQLValue*一致：nil必须输出NULL，否则布尔列的NULL在导出/迁移链路中被静默改写为false
	assert.Equal(t, "NULL", SQLValueBool(nil))
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
	// bitValuer统一返回int64（多字节BIT按大端合成数值），SQLValueNumeric以%v格式化生成SQL
	assert.Equal(t, int64(1), v.Value())

	*ptr = []byte{0}
	assert.Equal(t, int64(0), v.Value())

	// 多字节BIT(N)：driver按大端字节序返回，如BIT(16)的0x0102应合成为258
	*ptr = []byte{1, 2}
	assert.Equal(t, int64(258), v.Value())
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

// TestSplitColumnTypeBase 列类型串拆分：时间默认值需要列的小数秒精度、数值类型需要精度，
// 非纯数字参数（enum('a','b')）与无括号形态必须返回hasParam=false而非误读出数值
func TestSplitColumnTypeBase(t *testing.T) {
	kases := []struct {
		columnType string
		base       string
		num        int
		hasParam   bool
	}{
		{"datetime(3)", "datetime", 3, true},
		{"timestamp(6)", "timestamp", 6, true},
		{"DATETIME", "datetime", 0, false},
		{" datetime(2) ", "datetime", 2, true},
		{"decimal(20,6)", "decimal", 20, true},
		{"varchar(255)", "varchar", 255, true},
		{"bigint unsigned", "bigint unsigned", 0, false},
		{"enum('a','b')", "enum", 0, false},
		{"int[3]", "int[3]", 0, false},
		{"numeric()", "numeric", 0, false},
		{"", "", 0, false},
	}

	for _, k := range kases {
		base, num, hasParam := SplitColumnTypeBase(k.columnType)
		assert.Equal(t, k.base, base, "columnType=%q", k.columnType)
		assert.Equal(t, k.num, num, "columnType=%q", k.columnType)
		assert.Equal(t, k.hasParam, hasParam, "columnType=%q", k.columnType)
	}
}
