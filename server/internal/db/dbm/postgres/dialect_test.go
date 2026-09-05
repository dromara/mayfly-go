package postgres

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestSQLGenerator(dbType dbi.DbType) *SQLGenerator {
	// 触发postgres列类型与转换器注册（真实链路中GetMeta由连接创建时调用）
	dbi.GetMeta(DbTypePostgres)
	return &SQLGenerator{
		dialect: &PgsqlDialect{},
		dc:      &dbi.DbConn{Info: &dbi.DbInfo{Type: dbType}},
	}
}

func TestPostgresGenTableDDL(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)

	table := dbi.Table{TableName: "t_user", TableComment: "用户表"}
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int8", IsPrimaryKey: true, Nullable: false, ColumnComment: "主键"},
		{ColumnName: "name", DataType: "varchar", CharMaxLength: 50, Nullable: true},
		{ColumnName: "created_at", DataType: "timestamp", AutoIncrement: false, Nullable: true, ColumnDefault: "now()"},
	}

	sqls := gen.GenTableDDL(table, columns, true)
	// [drop, create, 表注释, 列注释]
	assert.Len(t, sqls, 4)
	assert.Equal(t, "DROP TABLE IF EXISTS \"t_user\"", sqls[0])
	assert.Contains(t, sqls[1], "CREATE TABLE \"t_user\" (\n")
	assert.Contains(t, sqls[1], " \"id\" int8 NOT NULL")
	assert.Contains(t, sqls[1], " \"name\" varchar(50)")
	// 函数默认值（now()）因跨库函数不兼容被忽略，不生成DEFAULT子句
	assert.NotContains(t, sqls[1], "DEFAULT")
	assert.Contains(t, sqls[1], "PRIMARY KEY (\"id\")")
	assert.Equal(t, "COMMENT ON TABLE \"t_user\" IS '用户表'", sqls[2])
	assert.Equal(t, "COMMENT ON COLUMN \"t_user\".\"id\" IS '主键'", sqls[3])
}

func TestPostgresGenTableDDL_AutoIncrementToSerial(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int4", AutoIncrement: true},
		{ColumnName: "big_id", DataType: "int8", AutoIncrement: true},
		{ColumnName: "small_id", DataType: "int2", AutoIncrement: true},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[0], " \"id\" serial NOT NULL")
	assert.Contains(t, sqls[0], " \"big_id\" bigserial NOT NULL")
	assert.Contains(t, sqls[0], " \"small_id\" smallserial NOT NULL")
}

func TestPostgresGenTableDDL_CommentQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)
	columns := []dbi.Column{
		{ColumnName: "name", DataType: "varchar", CharMaxLength: 50, Nullable: true, ColumnComment: "含'单引号'注释"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[len(sqls)-1], "IS '含''单引号''注释'")
}

// 默认值含单引号时需双写转义，否则 DDL 语法错误或注入
func TestPostgresGenTableDDL_DefaultQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)
	columns := []dbi.Column{
		{ColumnName: "status", DataType: "varchar", CharMaxLength: 10, Nullable: true, ColumnDefault: "it's"},
		{ColumnName: "num", DataType: "int8", Nullable: true, ColumnDefault: "0"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[0], " DEFAULT 'it''s'")
	// 数字默认值不走引号包裹，原样输出
	assert.Contains(t, sqls[0], " DEFAULT 0")
}

func TestPostgresGenIndexDDL(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)
	table := dbi.Table{TableName: "t1"}

	sqls := gen.GenIndexDDL(table, []dbi.Index{
		{IndexName: "idx_name", ColumnName: "name", IsUnique: true, IndexComment: "姓名索引"},
	})
	assert.Len(t, sqls, 3)
	// 伪连接（无schema）时不带schema前缀
	assert.Equal(t, "DROP INDEX IF EXISTS \"idx_name\"", sqls[0])
	assert.Equal(t, "CREATE unique INDEX \"idx_name\" ON \"t1\"(\"name\")", sqls[1])
	assert.Equal(t, "COMMENT ON INDEX \"idx_name\" IS '姓名索引'", sqls[2])
}

func TestPostgresGenInsert_None(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int8"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}, {2, "it's"}}

	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, "INSERT INTO \"t1\" (\"id\", \"name\") VALUES \n(1, 'a'),\n(2, 'it''s')", sqls[0])

	// Update策略但无元信息：不带on conflict后缀（退化为直接插入）
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Equal(t, sqls[0], "INSERT INTO \"t1\" (\"id\", \"name\") VALUES \n(1, 'a'),\n(2, 'it''s')")
}

func TestPostgresGenInsert_Ignore(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int8"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}}

	// on conflict do nothing 无需指定冲突列，可匹配任意唯一约束
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, "INSERT INTO \"t1\" (\"id\", \"name\") VALUES \n(1, 'a') \n on conflict do nothing", sqls[0])
}

func TestPostgresGenInsert_OnConflictUpdate(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int8"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}}

	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 1)
	// 单个on conflict子句，且不更新冲突键列自身
	expected := "INSERT INTO \"t1\" (\"id\", \"name\") VALUES \n(1, 'a')" +
		" \n on conflict (\"id\") do update set name = excluded.name \n"
	assert.Equal(t, expected, sqls[0])
}

func TestPostgresGenInsert_OnConflictUpdateAllUniqueDegenerate(t *testing.T) {
	gen := newTestSQLGenerator(DbTypePostgres)

	// 所有列均为冲突键列：退化为do nothing
	columns := []dbi.Column{{ColumnName: "id", DataType: "int8"}}
	values := [][]any{{1}}

	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, &dbi.TargetTableMeta{UniqueColumns: []string{"id"}})
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], " on conflict do nothing")
}

func TestPostgresGenInsert_Gauss(t *testing.T) {
	gen := newTestSQLGenerator(DbTypeGauss)

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int8"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}}

	// 高斯db的ignore策略
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, nil)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], " \n ON DUPLICATE KEY UPDATE NOTHING")

	// 高斯db的update策略：ON DUPLICATE KEY UPDATE，排除唯一键列自身
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], " \n ON DUPLICATE KEY UPDATE name = excluded.name")

	// 高斯db的update策略但无元信息：无后缀
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Len(t, sqls, 1)
	assert.NotContains(t, sqls[0], "ON DUPLICATE")
}
