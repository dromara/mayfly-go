package clickhouse

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestSQLGenerator() *ClickHouseSQLGenerator {
	// 触发clickhouse列类型与转换器注册（真实链路中GetMeta由连接创建时调用）
	dbi.GetMeta(DbTypeClickHouse)
	return &ClickHouseSQLGenerator{dialect: &ClickHouseDialect{}}
}

func TestClickHouseGenTableDDL(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "UInt64"},
		{ColumnName: "name", DataType: "String", ColumnComment: "姓'名'"},
	}

	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, true)
	assert.Len(t, sqls, 2)
	assert.Equal(t, "DROP TABLE IF EXISTS `t1`", sqls[0])
	assert.Contains(t, sqls[1], "CREATE TABLE `t1` (\n")
	assert.Contains(t, sqls[1], "  `id` UInt64")
	assert.Contains(t, sqls[1], "  `name` String COMMENT '姓''名'''")
	// 无主键时退化为tuple()排序
	assert.Contains(t, sqls[1], ") ENGINE = MergeTree() ORDER BY tuple()")
}

// Nullable包装、主键排序键、Decimal参数化（迁移异构库到clickhouse的核心DDL正确性）
func TestClickHouseGenTableDDL_NullablePrimaryKeyDecimal(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "UInt64", IsPrimaryKey: true},
		// 可空列必须包装Nullable，否则插入NULL会报错；主键列不可为Nullable
		{ColumnName: "name", DataType: "String", Nullable: true},
		{ColumnName: "score", DataType: "Decimal", NumPrecision: 10, NumScale: 2},
		{ColumnName: "count", DataType: "Decimal", NumPrecision: 10},
	}

	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Len(t, sqls, 1)
	ddl := sqls[0]
	assert.Contains(t, ddl, "  `id` UInt64")
	assert.Contains(t, ddl, "  `name` Nullable(String)")
	assert.Contains(t, ddl, "  `score` Decimal(10,2)")
	// NumScale为0时补齐S，避免生成非法的Decimal(10)
	assert.Contains(t, ddl, "  `count` Decimal(10, 0)")
	// 有主键时按主键排序，且主键列未被Nullable包装
	assert.Contains(t, ddl, ") ENGINE = MergeTree() ORDER BY (`id`)")
}

// 同库dump：源类型自带Nullable前缀时不重复包装
func TestClickHouseGenTableDDL_NullableIdempotent(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "name", DataType: "Nullable(String)", Nullable: true},
	}

	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[0], "  `name` Nullable(String)")
}

func TestClickHouseGenInsert_Simple(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "UInt64"},
		{ColumnName: "name", DataType: "String"},
	}
	values := [][]any{{uint64(1), "a"}, {uint64(2), "it's"}}

	// None策略：直接插入
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, "INSERT INTO `t1` (`id`, `name`) VALUES (1, 'a'), (2, 'it''s')", sqls[0])

	// Ignore策略：退化为直接插入（依赖MergeTree引擎去重）
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, sqls[0], "INSERT INTO `t1` (`id`, `name`) VALUES (1, 'a'), (2, 'it''s')")

	// Update策略但无唯一键元信息：退化为直接插入
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, &dbi.TargetTableMeta{})
	assert.Len(t, sqls, 1)

	// 空values：返回空切片
	assert.Empty(t, gen.GenInsert("t1", columns, [][]any{}, dbi.DuplicateStrategyNone, nil))
}

func TestClickHouseGenInsert_UpdateWithUniqueColumns(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "UInt64"},
		{ColumnName: "name", DataType: "String"},
	}
	values := [][]any{{uint64(1), "a"}, {uint64(2), "b"}}

	// Update策略且有唯一键：先删除冲突行再插入
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 2)
	assert.Equal(t, "ALTER TABLE `t1` DELETE WHERE (`id`) IN ((1), (2))", sqls[0])
	assert.Equal(t, "INSERT INTO `t1` (`id`, `name`) VALUES (1, 'a'), (2, 'b')", sqls[1])
}

func TestClickHouseGenInsert_UpdateMultiUniqueColumns(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "k1", DataType: "UInt64"},
		{ColumnName: "k2", DataType: "UInt64"},
		{ColumnName: "val", DataType: "String"},
	}
	values := [][]any{{uint64(1), uint64(2), "x"}}

	// 联合唯一键：删除条件为元组IN
	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"k1", "k2"}}
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 2)
	assert.Equal(t, "ALTER TABLE `t1` DELETE WHERE (`k1`, `k2`) IN ((1, 2))", sqls[0])
}

func TestClickHouseGenInsert_UniqueColumnsNotInInsert(t *testing.T) {
	gen := newTestSQLGenerator()

	// 唯一列不在插入列中：无法构建删除条件，退化为直接插入
	columns := []dbi.Column{
		{ColumnName: "name", DataType: "String"},
	}
	values := [][]any{{"a"}}

	meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, meta)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "INSERT INTO")
}
