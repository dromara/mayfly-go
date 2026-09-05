package dm

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestSQLGenerator() *SQLGenerator {
	// 触发达梦列类型与转换器注册（真实链路中GetMeta由连接创建时调用）
	dbi.GetMeta(DbTypeDM)
	return &SQLGenerator{Dialect: &DMDialect{}}
}

func TestDmGenTableDDL(t *testing.T) {
	gen := newTestSQLGenerator()

	table := dbi.Table{TableName: "t_user", TableComment: "用户表"}
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "INT", IsPrimaryKey: true, Nullable: false, ColumnComment: "主键"},
		{ColumnName: "name", DataType: "VARCHAR", CharMaxLength: 50, Nullable: true},
	}

	sqls := gen.GenTableDDL(table, columns, true)
	// [drop, create, 表注释, 列注释]
	assert.Len(t, sqls, 4)
	assert.Equal(t, "drop table if exists \"t_user\"", sqls[0])
	assert.Contains(t, sqls[1], "create table \"t_user\" (")
	assert.Contains(t, sqls[1], " \"id\" INT NOT NULL")
	assert.Contains(t, sqls[1], " \"name\" VARCHAR(50)")
	assert.Contains(t, sqls[1], "PRIMARY KEY (\"id\")")
	assert.Equal(t, "comment on table \"t_user\" is '用户表'", sqls[2])
	assert.Equal(t, "comment on column \"t_user\".\"id\" is '主键'", sqls[3])
}

func TestDmGenTableDDL_CommentQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "name", DataType: "VARCHAR", CharMaxLength: 50, Nullable: true, ColumnComment: "含'单引号'注释"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[len(sqls)-1], "is '含''单引号''注释'")
}

// 默认值含单引号时需双写转义，否则 DDL 语法错误或注入
func TestDmGenTableDDL_DefaultQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		// 单个不成对单引号：不命中 dm 的“已带引号包裹”分支，应走转义路径
		{ColumnName: "status", DataType: "VARCHAR", CharMaxLength: 10, Nullable: true, ColumnDefault: "it's"},
		// 已带引号包裹的默认值由元数据原样下发，不再转义包裹
		{ColumnName: "flag", DataType: "VARCHAR", CharMaxLength: 10, Nullable: true, ColumnDefault: "'Y'"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[0], " DEFAULT 'it''s'")
	assert.Contains(t, sqls[0], " DEFAULT 'Y'")
}

func TestDmGenIndexDDL(t *testing.T) {
	gen := newTestSQLGenerator()
	table := dbi.Table{TableName: "t1"}

	sqls := gen.GenIndexDDL(table, []dbi.Index{
		{IndexName: "idx_name", ColumnName: "name", IsUnique: false},
		{IndexName: "uk_id", ColumnName: "id", IsUnique: true},
	})
	assert.Len(t, sqls, 2)
	assert.Equal(t, "create  index \"idx_name\" on \"t1\"(\"name\")", sqls[0])
	assert.Equal(t, "create unique index \"uk_id\" on \"t1\"(\"id\")", sqls[1])
}

func TestDmGenInsert_Simple(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "INT"},
		{ColumnName: "name", DataType: "VARCHAR"},
	}
	values := [][]any{{1, "a"}, {2, "it's"}}

	// None策略：直接插入（达梦逐条insert）
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 2)
	assert.Equal(t, "insert into \"t1\" (\"id\", \"name\") values (1, 'a')", sqls[0])
	assert.Equal(t, "insert into \"t1\" (\"id\", \"name\") values (2, 'it''s')", sqls[1])

	// Ignore策略且无元信息：退化为直接插入（由数据库唯一约束报错，不静默丢数据）
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, nil)
	assert.Len(t, sqls, 2)
	assert.Equal(t, sqls[0], "insert into \"t1\" (\"id\", \"name\") values (1, 'a')")

	// Update策略但目标表无唯一键信息：同样退化为直接插入
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, &dbi.TargetTableMeta{})
	assert.Len(t, sqls, 2)
}

func TestDmGenInsert_SimpleWithIdentity(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "INT", AutoIncrement: true},
		{ColumnName: "name", DataType: "VARCHAR"},
	}
	values := [][]any{{1, "a"}, {2, "b"}}

	// 含自增列：首尾包裹identity_insert开关语句
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 4)
	assert.Equal(t, "set identity_insert \"t1\" on;", sqls[0])
	assert.Equal(t, "insert into \"t1\" (\"id\", \"name\") values (1, 'a')", sqls[1])
	assert.Equal(t, "insert into \"t1\" (\"id\", \"name\") values (2, 'b')", sqls[2])
	// 结尾为identity off语句
	assert.Equal(t, "set identity_insert \"t1\" off;", sqls[3])
}

func TestDmGenInsert_Merge(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "INT"},
		{ColumnName: "name", DataType: "VARCHAR"},
	}
	values := [][]any{{1, "a"}}

	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 1)

	expected := "MERGE INTO \"t1\" T1 USING (SELECT ? id,? name FROM dual) T2 ON ( T1.\"id\" = T2.\"id\" )" +
		"WHEN NOT MATCHED THEN INSERT (id,name) VALUES (T2.id,T2.name)" +
		"WHEN MATCHED THEN UPDATE SET T1.name = T2.name"
	assert.Equal(t, expected, sqls[0])
}

func TestDmGenInsert_MergeAllUniqueColumnsDegenerate(t *testing.T) {
	gen := newTestSQLGenerator()

	// 仅一列且为唯一键列：无法生成update子句，退化为直接插入
	columns := []dbi.Column{{ColumnName: "id", DataType: "INT"}}
	values := [][]any{{1}, {2}}

	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, &dbi.TargetTableMeta{UniqueColumns: []string{"id"}})
	assert.Len(t, sqls, 2)
	assert.Contains(t, sqls[0], "insert into \"t1\"")
}

func TestDmGenInsert_MergeExcludeIdentityColumns(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "INT", AutoIncrement: true},
		{ColumnName: "name", DataType: "VARCHAR"},
	}
	values := [][]any{{1, "a"}}

	// id为自增列：merge的insert子句需排除自增列
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}, IdentityColumns: []string{"id"}}
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "INSERT (name) VALUES (T2.name)")
	assert.Contains(t, sqls[0], "WHEN MATCHED THEN UPDATE SET T1.name = T2.name")
}

func TestDmGenInsert_MergeMultiUniqueColumns(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "a", DataType: "INT"},
		{ColumnName: "b", DataType: "INT"},
		{ColumnName: "val", DataType: "VARCHAR"},
	}
	values := [][]any{{1, 2, "x"}}

	// 联合唯一键：ON条件用OR连接（保持原有语义）
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"a", "b"}}
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "ON ( T1.\"a\" = T2.\"a\" ) OR ( T1.\"b\" = T2.\"b\" )")
}
