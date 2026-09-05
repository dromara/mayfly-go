package oracle

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestSQLGenerator() *SQLGenerator {
	// 触发oracle列类型与转换器注册（真实链路中GetMeta由连接创建时调用）
	dbi.GetMeta(DbTypeOracle)
	return &SQLGenerator{Dialect: &OracleDialect{}}
}

func TestOracleGenTableDDL(t *testing.T) {
	gen := newTestSQLGenerator()

	table := dbi.Table{TableName: "T_USER", TableComment: "用户表"}
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "NUMBER", IsPrimaryKey: true, Nullable: false, ColumnComment: "主键"},
		{ColumnName: "name", DataType: "VARCHAR2", CharMaxLength: 50, Nullable: true},
	}

	sqls := gen.GenTableDDL(table, columns, false)
	// [create, 表注释, 列注释]
	assert.Len(t, sqls, 3)
	assert.Contains(t, sqls[0], "CREATE TABLE \"T_USER\" ( \n")
	assert.Contains(t, sqls[0], " \"id\" NUMBER NOT NULL")
	assert.Contains(t, sqls[0], " \"name\" VARCHAR2(50)")
	assert.Contains(t, sqls[0], "PRIMARY KEY (\"id\")")
	assert.Equal(t, "COMMENT ON TABLE \"T_USER\" is '用户表'", sqls[1])
	assert.Equal(t, "COMMENT ON COLUMN \"T_USER\".\"id\" IS '主键'", sqls[2])
}

func TestOracleGenTableDDL_CommentQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "name", DataType: "VARCHAR2", CharMaxLength: 50, Nullable: true, ColumnComment: "含'单引号'注释"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "T1"}, columns, false)
	assert.Contains(t, sqls[len(sqls)-1], "IS '含''单引号''注释'")
}

func TestOracleGenIndexDDL(t *testing.T) {
	gen := newTestSQLGenerator()
	table := dbi.Table{TableName: "T1"}

	sqls := gen.GenIndexDDL(table, []dbi.Index{
		{IndexName: "idx_name", ColumnName: "name", IsUnique: false},
		{IndexName: "uk_id", ColumnName: "id", IsUnique: true},
	})
	assert.Len(t, sqls, 2)
	assert.Equal(t, "CREATE  INDEX \"idx_name\" ON \"T1\"(\"name\")", sqls[0])
	assert.Equal(t, "CREATE unique INDEX \"uk_id\" ON \"T1\"(\"id\")", sqls[1])
}

func TestOracleGenInsert_Simple(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "NUMBER"},
		{ColumnName: "name", DataType: "VARCHAR2"},
	}
	values := [][]any{{1, "a"}, {2, "it's"}}

	// None策略：oracle逐条insert，columnStr带一层括号（不允许双括号）
	sqls := gen.GenInsert("T1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 2)
	assert.Equal(t, " insert into \"T1\" (\"id\", \"name\") values (1, 'a')", sqls[0])
	assert.Equal(t, " insert into \"T1\" (\"id\", \"name\") values (2, 'it''s')", sqls[1])
	// 防止双括号回归
	assert.NotContains(t, sqls[0], "((")
}

func TestOracleGenInsert_SimpleWithIdentityPrefix(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "NUMBER", AutoIncrement: true},
	}
	values := [][]any{{1}}

	// 含自增列：每条insert前带identity_insert语句
	sqls := gen.GenInsert("T1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, "set identity_insert \"T1\" on; insert into \"T1\" (\"id\") values (1)", sqls[0])
}

func TestOracleGenInsert_Merge(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "NUMBER"},
		{ColumnName: "name", DataType: "VARCHAR2"},
	}
	values := [][]any{{1, "a"}}

	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	sqls := gen.GenInsert("T1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 1)

	expected := "MERGE INTO \"T1\" T1 USING (SELECT ? id,? name FROM dual) T2 ON ( T1.\"id\" = T2.\"id\" )" +
		"WHEN NOT MATCHED THEN INSERT (id,name) VALUES (T2.id,T2.name)" +
		"WHEN MATCHED THEN UPDATE SET T1.name = T2.name"
	assert.Equal(t, expected, sqls[0])
}

func TestOracleGenInsert_Degenerate(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{{ColumnName: "id", DataType: "NUMBER"}}
	values := [][]any{{1}, {2}}

	// 无元信息退化为直接插入
	sqls := gen.GenInsert("T1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Len(t, sqls, 2)
	assert.Contains(t, sqls[0], "insert into \"T1\"")

	// 全列为唯一键列退化为直接插入
	sqls = gen.GenInsert("T1", columns, values, dbi.DuplicateStrategyUpdate, &dbi.TargetTableMeta{UniqueColumns: []string{"id"}})
	assert.Len(t, sqls, 2)
	assert.Contains(t, sqls[0], "insert into \"T1\"")
}

func TestOracleGenInsert_MergeExcludeIdentityColumns(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "NUMBER", AutoIncrement: true},
		{ColumnName: "name", DataType: "VARCHAR2"},
	}
	values := [][]any{{1, "a"}}

	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}, IdentityColumns: []string{"id"}}
	sqls := gen.GenInsert("T1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "INSERT (name) VALUES (T2.name)")
}
