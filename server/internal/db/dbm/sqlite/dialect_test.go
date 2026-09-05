package sqlite

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestSQLGenerator() *SQLGenerator {
	// 触发sqlite列类型与转换器注册（真实链路中GetMeta由连接创建时调用）
	dbi.GetMeta(DbTypeSqlite)
	return &SQLGenerator{dialect: &SqliteDialect{}}
}

func TestSqliteGenTableDDL(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "integer", IsPrimaryKey: true, AutoIncrement: true, Nullable: false},
		{ColumnName: "name", DataType: "text", Nullable: true, ColumnDefault: "匿名"},
	}

	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, true)
	assert.Len(t, sqls, 2)
	assert.Equal(t, "DROP TABLE IF EXISTS \"t1\"", sqls[0])
	assert.Contains(t, sqls[1], "CREATE TABLE \"t1\" (\n")
	// 主键自增
	assert.Contains(t, sqls[1], " \"id\" integer PRIMARY KEY AUTOINCREMENT NOT NULL")
	// 字符串默认值加引号
	assert.Contains(t, sqls[1], " DEFAULT '匿名'")

	// 非主键列不走PRIMARY KEY逻辑
	columns2 := []dbi.Column{{ColumnName: "name", DataType: "text", Nullable: false}}
	sqls2 := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns2, false)
	assert.Contains(t, sqls2[0], " \"name\" text NOT NULL")
}

// 默认值含单引号时需双写转义，否则 DDL 语法错误或注入
func TestSqliteGenTableDDL_DefaultQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "status", DataType: "text", Nullable: true, ColumnDefault: "it's"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[0], " DEFAULT 'it''s'")
}

func TestSqliteGenIndexDDL(t *testing.T) {
	gen := newTestSQLGenerator()
	table := dbi.Table{TableName: "t1"}

	sqls := gen.GenIndexDDL(table, []dbi.Index{
		{IndexName: "idx_name", ColumnName: "name", IsUnique: true},
	})
	assert.Len(t, sqls, 2)
	assert.Equal(t, "DROP INDEX IF EXISTS \"idx_name\"", sqls[0])
	assert.Equal(t, "CREATE unique INDEX \"idx_name\" ON \"t1\" (\"name\") ", sqls[1])
}

func TestSqliteGenInsert_None(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "integer"},
		{ColumnName: "name", DataType: "text"},
	}
	values := [][]any{{1, "a"}, {2, "it's"}}

	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, "INSERT INTO \"t1\" (\"id\", \"name\") VALUES \n(1, 'a'),\n(2, 'it''s')", sqls[0])
}

func TestSqliteGenInsert_IgnoreAndReplace(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "integer"},
		{ColumnName: "name", DataType: "text"},
	}
	values := [][]any{{1, "a"}}

	// Ignore策略：PRAGMA开关 + insert or ignore
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, nil)
	assert.Len(t, sqls, 3)
	assert.Equal(t, "PRAGMA foreign_keys = false", sqls[0])
	assert.Equal(t, "PRAGMA foreign_keys = true", sqls[1])
	assert.Contains(t, sqls[2], "insert or ignore into \"t1\" (\"id\", \"name\") VALUES \n(1, 'a')")

	// Update策略：insert or replace
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Len(t, sqls, 3)
	assert.Contains(t, sqls[2], "insert or replace into \"t1\"")
}
