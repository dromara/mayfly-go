package dbi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
