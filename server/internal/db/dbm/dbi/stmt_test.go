package dbi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 为测试专用dbType注册数据类型：未注册时全部退化到DefaultDbDataType（字符串类型），
// 测不出数字类型“无引号输出”与字符串类型“强制引号”的分叉行为
func init() {
	registerColumnDbDataTypes(DbType("test-stmt-db"),
		NewDbDataType("int8", DTInt64).WithCT(CTInt8),
		NewDbDataType("varchar", DTString).WithCT(CTVarchar),
	)
}

func TestGenInsertSqlColumnAndValues(t *testing.T) {
	dialect := newStubDialect() // DefaultQuoter: "xx"
	dbType := DbType("test-stmt-db")

	columns := []Column{
		{ColumnName: "id", DataType: "int8"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{
		{1, "it's"},
		{2, nil},
	}

	columnsStr, valuesStrs := GenInsertSqlColumnAndValues(dialect, dbType, columns, values)

	assert.Equal(t, `("id", "name")`, columnsStr)
	assert.Len(t, valuesStrs, 2)
	assert.Equal(t, `(1, 'it''s')`, valuesStrs[0])
	// nil应转成NULL而非'NULL'字符串
	assert.Equal(t, `(2, NULL)`, valuesStrs[1])
}

// 行值数与列数不一致时必须快速失败，避免columnTypes越界panic或静默错位生成错误SQL
func TestGenInsertSqlColumnAndValues_ColumnValueMismatch(t *testing.T) {
	dialect := newStubDialect()
	dbType := DbType("test-stmt-db")

	columns := []Column{
		{ColumnName: "id", DataType: "int8"},
		{ColumnName: "name", DataType: "varchar"},
	}

	// 行值少于列数
	assert.PanicsWithValue(t,
		"gen insert sql for db [test-stmt-db]: row has 1 values but 2 columns are provided",
		func() { GenInsertSqlColumnAndValues(dialect, dbType, columns, [][]any{{1}}) })

	// 行值多于列数
	assert.PanicsWithValue(t,
		"gen insert sql for db [test-stmt-db]: row has 3 values but 2 columns are provided",
		func() { GenInsertSqlColumnAndValues(dialect, dbType, columns, [][]any{{1, "a", 2}}) })
}

func TestGenCommonInsert(t *testing.T) {
	dialect := newStubDialect()
	dbType := DbType("test-stmt-db")

	columns := []Column{
		{ColumnName: "id", DataType: "int8"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{
		{1, "a"},
		{2, "b'c"},
	}

	sql := GenCommonInsert(dialect, dbType, "t1", columns, values)
	expected := "INSERT INTO \"t1\" (\"id\", \"name\") VALUES \n(1, 'a'),\n(2, 'b''c')"
	assert.Equal(t, expected, sql)
}

func TestGenCommonInsert_EmptyValues(t *testing.T) {
	dialect := newStubDialect()
	columns := []Column{{ColumnName: "id", DataType: "int8"}}

	// 空values应生成仅有列信息、无values的语句（调用方应自行判断不执行）
	sql := GenCommonInsert(dialect, DbType("test-stmt-db"), "t1", columns, [][]any{})
	assert.Equal(t, "INSERT INTO \"t1\" (\"id\") VALUES \n", sql)
}
