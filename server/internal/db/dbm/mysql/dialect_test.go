package mysql

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
	"mayfly-go/pkg/utils/collx"
)

func newTestSQLGenerator() *SQLGenerator {
	// 触发mysql列类型与转换器注册（真实链路中GetMeta由连接创建时调用）
	dbi.GetMeta(DbTypeMysql)
	return &SQLGenerator{Dialect: &MysqlDialect{}}
}

func TestMysqlGenTableDDL(t *testing.T) {
	gen := newTestSQLGenerator()

	table := dbi.Table{TableName: "t_user", TableComment: "用户表"}
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true, AutoIncrement: true, Nullable: false, ColumnComment: "自增主键"},
		{ColumnName: "name", DataType: "varchar", CharMaxLength: 50, Nullable: true, ColumnComment: "姓名"},
	}

	sqls := gen.GenTableDDL(table, columns, true)
	assert.Len(t, sqls, 2)
	assert.Equal(t, "DROP TABLE IF EXISTS `t_user`", sqls[0])
	createSql := sqls[1]

	assert.Contains(t, createSql, "CREATE TABLE `t_user` (\n")
	// 列定义：非空自增主键
	assert.Contains(t, createSql, "`id` int NOT NULL AUTO_INCREMENT COMMENT '自增主键'")
	// varchar列带长度
	assert.Contains(t, createSql, "`name` varchar(50) COMMENT '姓名'")
	// 主键
	assert.Contains(t, createSql, "PRIMARY KEY (id)")
	// 表注释单引号转义
	assert.Contains(t, createSql, " COMMENT '用户表'")

	// 不删除重建时不含DROP
	sqls2 := gen.GenTableDDL(table, columns, false)
	assert.Len(t, sqls2, 1)
	assert.NotContains(t, sqls2[0], "DROP TABLE")
}

func TestMysqlGenTableDDL_CommentQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "name", DataType: "varchar", CharMaxLength: 50, Nullable: true, ColumnComment: "含'单引号'注释"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	// 注释中的单引号必须双写转义，否则SQL语法错误
	assert.Contains(t, sqls[0], "COMMENT '含''单引号''注释'")
}

// 默认值含单引号时需双写转义，否则 DDL 语法错误或注入
func TestMysqlGenTableDDL_DefaultQuoteEscape(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "status", DataType: "varchar", CharMaxLength: 10, Nullable: true, ColumnDefault: "it's"},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	assert.Contains(t, sqls[0], " DEFAULT 'it''s'")
}

func TestMysqlGenIndexDDL(t *testing.T) {
	gen := newTestSQLGenerator()
	table := dbi.Table{TableName: "t1"}

	sqls := gen.GenIndexDDL(table, []dbi.Index{
		{IndexName: "idx_name", ColumnName: "name", IsUnique: false},
		{IndexName: "uk_name", ColumnName: "name", IsUnique: true},
	})
	assert.Len(t, sqls, 2)
	assert.Equal(t, "ALTER TABLE `t1` ADD  INDEX `idx_name`(`name`) USING BTREE", sqls[0])
	assert.Equal(t, "ALTER TABLE `t1` ADD unique INDEX `uk_name`(`name`) USING BTREE", sqls[1])

	// 前缀索引
	sqls = gen.GenIndexDDL(table, []dbi.Index{
		{IndexName: "idx_name", ColumnName: "name", Extra: collx.Kvs(IndexSubPartKey, 10)},
	})
	assert.Equal(t, "ALTER TABLE `t1` ADD  INDEX `idx_name`(`name`(10)) USING BTREE", sqls[0])
}

func TestMysqlGenInsert(t *testing.T) {
	gen := newTestSQLGenerator()

	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int8"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}, {2, "it's"}}

	// 无冲突处理策略：标准INSERT
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, "INSERT INTO `t1` (`id`, `name`) VALUES \n(1, 'a'),\n(2, 'it''s')", sqls[0])

	// 忽略冲突：insert ignore into（mysql无需元信息）
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyIgnore, nil)
	assert.Len(t, sqls, 1)
	assert.True(t, len(sqls[0]) > 0 && sqls[0][:len("insert ignore into")] == "insert ignore into", "应为insert ignore into前缀: %s", sqls[0])
	assert.Contains(t, sqls[0], "`t1`")

	// 更新冲突：replace into（mysql用replace语义，无需TargetTableMeta）
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, nil)
	assert.Len(t, sqls, 1)
	assert.True(t, len(sqls[0]) > 0 && sqls[0][:len("replace into")] == "replace into", "应为replace into前缀: %s", sqls[0])

	// Update策略即使传入TargetTableMeta也使用replace into
	sqls = gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyUpdate, &dbi.TargetTableMeta{UniqueColumns: []string{"id"}})
	assert.Len(t, sqls, 1)
	assert.True(t, len(sqls[0]) > 0 && sqls[0][:len("replace into")] == "replace into", "应为replace into前缀: %s", sqls[0])
}

func TestMysqlGenInsert_ValueEscape(t *testing.T) {
	gen := newTestSQLGenerator()

	// mysql的varchar应使用反斜杠转义规则
	columns := []dbi.Column{
		{ColumnName: "name", DataType: "varchar"},
		{ColumnName: "remark", DataType: "text"},
	}
	values := [][]any{{`a\b`, "it's"}}

	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Contains(t, sqls[0], `'a\\b'`) // 反斜杠双写
	assert.Contains(t, sqls[0], `'it''s'`)
}
