package mssql

import (
	"mayfly-go/internal/db/dbm/dbi"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestSQLGenerator() *SQLGenerator {
	// 触发mssql列类型与转换器注册（真实链路中GetMeta由连接创建时调用）
	meta := dbi.GetMeta(DbTypeMssql)
	return &SQLGenerator{dc: &dbi.DbConn{Info: &dbi.DbInfo{Type: DbTypeMssql, Meta: meta}}}
}

func TestMssqlQuoteTableName(t *testing.T) {
	gen := newTestSQLGenerator()
	quote := gen.dc.GetDialect().Quoter().Quote

	// 伪连接场景（Database为空）schema为空，不允许生成 [].[table] 形式的坏SQL
	assert.Equal(t, "[t1]", gen.quoteTableName(quote, "t1"))

	// database/schema模式下带schema前缀
	gen.dc.Info.Database = "db1/dbo"
	assert.Equal(t, "[dbo].[t1]", gen.quoteTableName(quote, "t1"))
}

func TestMssqlGenTableDDL(t *testing.T) {
	gen := newTestSQLGenerator()

	table := dbi.Table{TableName: "t_user", TableComment: "用户表"}
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true, AutoIncrement: true, Nullable: false, ColumnComment: "主键"},
		{ColumnName: "name", DataType: "varchar", CharMaxLength: 50, Nullable: true},
	}

	sqls := gen.GenTableDDL(table, columns, true)
	// [drop, create, 表注释, 列注释]
	assert.Len(t, sqls, 4)
	assert.Equal(t, "DROP TABLE IF EXISTS [t_user]", sqls[0])
	assert.Contains(t, sqls[1], "CREATE TABLE [t_user] (\n")
	assert.Contains(t, sqls[1], " [id] int IDENTITY(1,1) NOT NULL")
	assert.Contains(t, sqls[1], " [name] varchar(50)")
	assert.Contains(t, sqls[1], "PRIMARY KEY CLUSTERED ([id])")
	assert.Equal(t, "EXECUTE sp_addextendedproperty N'MS_Description', N'用户表', N'SCHEMA', N'', N'TABLE', N't_user'", sqls[2])
	assert.Equal(t, "EXECUTE sp_addextendedproperty N'MS_Description', N'主键', N'SCHEMA', N'', N'TABLE', N't_user', N'COLUMN', N'id'", sqls[3])
}

func TestMssqlGenTableDDL_CommentQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "name", DataType: "varchar", CharMaxLength: 50, Nullable: true, ColumnComment: "含'单引号'注释"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[len(sqls)-1], "N'含''单引号''注释'")
}

// 默认值含单引号时需双写转义，否则 DDL 语法错误或注入
func TestMssqlGenTableDDL_DefaultQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "status", DataType: "varchar", CharMaxLength: 10, Nullable: true, ColumnDefault: "it's"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[0], " DEFAULT 'it''s'")
}

func TestMssqlGenIndexDDL(t *testing.T) {
	gen := newTestSQLGenerator()
	table := dbi.Table{TableName: "t1"}

	sqls := gen.GenIndexDDL(table, []dbi.Index{
		{IndexName: "idx_name", ColumnName: "name", IsUnique: true},
	})
	assert.Len(t, sqls, 1)
	assert.Equal(t, "create unique NONCLUSTERED index [idx_name] on [t1]([name])", sqls[0])
}

func TestMssqlGenInsert_Simple(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}, {2, "it's"}}

	// None策略：字面值插入（值转义由各类型的SQLValue处理）
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, " insert into [t1] ([id], [name]) VALUES \n(1, 'a'),\n(2, 'it''s')", sqls[0])

	// Ignore策略但无唯一键元信息：只生成insert，不生成IGNORE_DUP_KEY约束
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, sqls[0], " insert into [t1] ([id], [name]) VALUES \n(1, 'a'),\n(2, 'it''s')")
}

func TestMssqlGenInsert_IgnoreWithMeta(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}}

	// Ignore策略且有唯一键元信息：前加IGNORE_DUP_KEY=ON约束，后加OFF恢复
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, meta)
	assert.Len(t, sqls, 3)
	assert.Equal(t, "ALTER TABLE [t1] ADD CONSTRAINT uniqueRows UNIQUE (id) WITH (IGNORE_DUP_KEY = ON)", sqls[0])
	assert.Contains(t, sqls[1], "insert into [t1]")
	assert.Equal(t, "ALTER TABLE [t1] ADD CONSTRAINT uniqueRows UNIQUE (id) WITH (IGNORE_DUP_KEY = OFF)", sqls[2])
}

func TestMssqlGenInsert_Merge(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}, {2, "b"}}

	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Len(t, sqls, 1)

	sql := sqls[0]
	// INSERT子句必须带VALUES关键字
	assert.Contains(t, sql, "WHEN NOT MATCHED THEN INSERT ([id],[name]) VALUES (T2.id,T2.name)")
	// 多行值：每行的select只包含本行的值，且以UNION ALL连接
	assert.Contains(t, sql, "USING (select 1 id, 'a' name UNION ALL select 2 id, 'b' name) T2")
	assert.Contains(t, sql, "MERGE INTO [t1] T1")
	assert.Contains(t, sql, "ON  T1.[id] = T2.[id] ")
	// 非自增列均生成update子句（含主键列自身，按columns顺序）
	assert.Contains(t, sql, "WHEN MATCHED THEN UPDATE SET T1.id = T2.id,T1.name = T2.name")
	assert.True(t, strings.HasSuffix(sql, "WHEN MATCHED THEN UPDATE SET T1.id = T2.id,T1.name = T2.name"))
}

func TestMssqlGenInsert_MergeWithoutPKDegenerate(t *testing.T) {
	gen := newTestSQLGenerator()

	// 无主键无法生成merge，退化为简单插入
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}}

	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "insert into [t1]")
}

func TestMssqlGenInsert_MergeWithIdentity(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true, AutoIncrement: true},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}}

	// 含自增列：merge前加SET IDENTITY_INSERT ... ON
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Len(t, sqls, 1)
	assert.True(t, strings.HasPrefix(sqls[0], "SET IDENTITY_INSERT [t1] ON MERGE INTO"))
}

func TestMssqlGenInsert_BatchSizeLimit(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int"},
		{ColumnName: "name", DataType: "varchar"},
	}
	// 每行2个参数，1500行=3000参数 > 2000，应拆分为多批
	values := make([][]any, 0, 1500)
	for i := 0; i < 1500; i++ {
		values = append(values, []any{i, "name"})
	}

	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	// 每批最多1000行（2000/2），1500行应拆为2批
	assert.Len(t, sqls, 2)
	totalRows := strings.Count(sqls[0], "'name')") + strings.Count(sqls[1], "'name')")
	assert.Equal(t, 1500, totalRows)
}
