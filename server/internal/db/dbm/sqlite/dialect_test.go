package sqlite

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// 非自增主键必须保留原列类型；仅AUTOINCREMENT才强制integer
// （原实现对所有主键无条件强制integer PRIMARY KEY，text主键等迁移DDL错误）
func TestSqliteGenTableDDL_PrimaryKeyType(t *testing.T) {
	gen := newTestSQLGenerator()

	// text单列主键：保留原类型，不带AUTOINCREMENT
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t_pk_text"}, []dbi.Column{
		{ColumnName: "code", DataType: "text", IsPrimaryKey: true, Nullable: false},
	}, false)
	assert.Contains(t, sqls[0], " \"code\" text PRIMARY KEY")
	assert.NotContains(t, sqls[0], "text PRIMARY KEY AUTOINCREMENT")

	// integer自增主键：强制integer + AUTOINCREMENT
	sqls = gen.GenTableDDL(dbi.Table{TableName: "t_pk_ai"}, []dbi.Column{
		{ColumnName: "id", DataType: "integer", IsPrimaryKey: true, AutoIncrement: true, Nullable: false},
	}, false)
	assert.Contains(t, sqls[0], " \"id\" integer PRIMARY KEY AUTOINCREMENT")
}

// 复合主键必须用表级PRIMARY KEY(...)声明：逐列内联PRIMARY KEY在sqlite下直接报
// “table has more than one primary key”，生成的DDL无法执行
func TestSqliteGenTableDDL_CompositePrimaryKey(t *testing.T) {
	gen := newTestSQLGenerator()
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t_cp"}, []dbi.Column{
		{ColumnName: "a", DataType: "text", IsPrimaryKey: true, Nullable: false},
		{ColumnName: "b", DataType: "integer", IsPrimaryKey: true, Nullable: false},
		{ColumnName: "c", DataType: "text", Nullable: true},
	}, false)
	assert.Equal(t, "CREATE TABLE \"t_cp\" (\n \"a\" text NOT NULL,\n \"b\" integer NOT NULL,\n \"c\" text,\nPRIMARY KEY (\"a\",\"b\")\n)", sqls[0])
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

	// Ignore策略：PRAGMA false + insert or ignore + PRAGMA true（恢复语句必须在INSERT之后，
	// 否则两条PRAGMA相邻执行互相抵消，INSERT时外键约束仍处于开启状态）
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, nil)
	assert.Len(t, sqls, 3)
	assert.Equal(t, "PRAGMA foreign_keys = false", sqls[0])
	assert.Contains(t, sqls[1], "insert or ignore into \"t1\" (\"id\", \"name\") VALUES \n(1, 'a')")
	assert.Equal(t, "PRAGMA foreign_keys = true", sqls[2])

	// Update策略：insert or replace
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Len(t, sqls, 3)
	assert.Contains(t, sqls[1], "insert or replace into \"t1\"")
}

// TestSqliteGenIndexDDL_SpecialNames 索引名/表名列名含引用符时必须双写转义：
// DROP INDEX 此前硬编码引号拼接原始索引名，含双引号的名称会提前闭合而生成非法SQL
func TestSqliteGenIndexDDL_SpecialNames(t *testing.T) {
	gen := newTestSQLGenerator()
	indexs := []dbi.Index{{IndexName: `idx"1`, ColumnName: "c_中文 列"}}

	sqls := gen.GenIndexDDL(dbi.Table{TableName: `t"1`}, indexs)
	require.Len(t, sqls, 2)
	assert.Equal(t, `DROP INDEX IF EXISTS "idx""1"`, sqls[0])
	assert.Contains(t, sqls[1], `"idx""1"`)
	assert.Contains(t, sqls[1], `ON "t""1" ("c_中文 列")`)
}

// TestSqliteAffinityTypeRegistered SQLite是弱类型库，列声明类型只决定存储亲和性，
// 常见外库风格类型名（INT/BIGINT/TIMESTAMP/DECIMAL/VARCHAR等）必须按官方亲和性规则注册，
// 否则未注册类型会回退为DefaultDbDataType（CTVarchar），使这些列迁移到强类型库时被静默改成varchar
func TestSqliteAffinityTypeRegistered(t *testing.T) {
	newTestSQLGenerator()

	tests := []struct {
		name string
		want dbi.CommonDbDataType
	}{
		{"int", dbi.CTInt8},
		{"INT", dbi.CTInt8},
		{"bigint", dbi.CTInt8},
		{"integer", dbi.CTInt8},
		{"tinyint", dbi.CTInt8},
		{"varchar", dbi.CTVarchar},
		{"nvarchar", dbi.CTVarchar},
		{"char", dbi.CTChar},
		{"text", dbi.CTText},
		{"clob", dbi.CTText},
		{"numeric", dbi.CTDecimal},
		{"decimal", dbi.CTDecimal},
		{"float", dbi.CTNumeric},
		{"double", dbi.CTNumeric},
		{"real", dbi.CTNumeric},
		{"timestamp", dbi.CTTimestamp},
		{"datetime", dbi.CTDateTime},
		{"date", dbi.CTDate},
		{"blob", dbi.CTBlob},
	}
	for _, tt := range tests {
		got := dbi.GetDbDataType(DbTypeSqlite, tt.name)
		require.NotNil(t, got, "sqlite类型 [%s] 未注册", tt.name)
		assert.NotEqual(t, dbi.DefaultDbDataType, got, "sqlite类型 [%s] 不得回退为默认字符串类型", tt.name)
		assert.Equal(t, tt.want, got.CommonType, "sqlite类型 [%s] 亲和性映射不符", tt.name)
	}
}
