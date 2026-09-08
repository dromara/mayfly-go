package mysql

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// 主键列名必须引用（元数据标识符可合法含空格/分号等）
	assert.Contains(t, createSql, "PRIMARY KEY (`id`)")
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

// 默认值元数据呈现形态随MySQL版本而异（8.0为原始值，5.7/MariaDB为带引号字面量），
// 两侧都必须无损往返回到目标DDL：旧实现按字符集剥除所有首尾单引号，
// 会使以引号结尾的默认值（end'）静默丢字符，且enum等类型的默认值裸拼直接语法错误
func TestMysqlGenTableDDL_ColumnDefault(t *testing.T) {
	kases := []struct {
		name     string
		dataType string
		defVal   string
		expected string
	}{
		{"8.0原始值-普通", "varchar", "abc", ` DEFAULT 'abc'`},
		{"8.0原始值-含单引号", "varchar", "it's", ` DEFAULT 'it''s'`},
		{"5.7字面量形态还原", "varchar", "'it''s'", ` DEFAULT 'it''s'`},
		// 已知限制：值本身首尾成对带单引号时（如 'quoted'），information_schema无法与5.7字面量
		// 形态区分，只能按字面量语义解读（社区通用做法），丢失的是一层书写引号而非内容
		{"5.7字面量形态-含引号值", "varchar", "'''quoted'''", ` DEFAULT '''quoted'''`},
		{"尾部单引号不丢字符", "varchar", "end'", ` DEFAULT 'end'''`},
		{"反斜杠双写", "varchar", `a\b`, ` DEFAULT 'a\\b'`},
		{"语句拼接不得逃逸引号", "varchar", "1; DROP TABLE t--", ` DEFAULT '1; DROP TABLE t--'`},
		{"空白默认值保留", "varchar", " ", ` DEFAULT ' '`},
		{"enum默认值必须引用", "enum", "active", ` DEFAULT 'active'`},
		{"set默认值必须引用", "set", "a,b", ` DEFAULT 'a,b'`},
		{"数字默认值不加引号", "bigint", "3", " DEFAULT 3"},
		{"timestamp函数默认值不加引号", "timestamp", "CURRENT_TIMESTAMP", " DEFAULT CURRENT_TIMESTAMP"},
		{"日期字面量默认值加引号", "datetime", "2020-01-01 00:00:00", ` DEFAULT '2020-01-01 00:00:00'`},
		// MySQL 8.0的COLUMN_DEFAULT去引号呈现，使 varchar DEFAULT 'concat(a,b)' 与裸函数调用同形；
		// 而字符串列的表达式默认值必须书写为(concat(a,b))形态，故裸形态必为内容，当作函数丢弃会让非空列丢默认值
		{"字符串列的函数形默认值按内容保留", "varchar", "concat(a,b)", ` DEFAULT 'concat(a,b)'`},
		{"非字符串列的函数默认值跨源不支持而省略", "int", "concat(a,b)", ""},
	}

	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			gen := newTestSQLGenerator()
			columns := []dbi.Column{{ColumnName: "c1", DataType: k.dataType, CharMaxLength: 10, Nullable: true, ColumnDefault: k.defVal}}
			sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
			if k.expected == "" {
				assert.NotContains(t, sqls[0], "DEFAULT")
				return
			}
			assert.Contains(t, sqls[0], k.expected)
		})
	}
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
		{ColumnName: "id", DataType: "bigint"},
		{ColumnName: "name", DataType: "varchar"},
	}
	values := [][]any{{1, "a"}, {2, "it's"}}

	// 无冲突处理策略：标准INSERT
	sqls := gen.GenInsert("t1", columns, values, dbi.DuplicateStrategyNone, nil)
	assert.Len(t, sqls, 1)
	assert.Equal(t, "INSERT INTO `t1` (`id`, `name`) VALUES \n(1, 'a'),\n(2, 'it''s')", sqls[0])

	// 未注册的数据类型退化到默认字符串类型：值必须输出为带引号的字面量，
	// 不可裸拼（原文无引号输出会使含空格/分号的值成为独立语句）
	sqls = gen.GenInsert("t1", []dbi.Column{{ColumnName: "id", DataType: "unknown_type"}}, [][]any{{int64(1)}}, dbi.DuplicateStrategyNone, nil)
	assert.Equal(t, "INSERT INTO `t1` (`id`) VALUES \n('1')", sqls[0])

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

// TestMysqlGenTableDDL_SpecialPkColumnName 主键列名含空格/分号时必须引用：
// 此前直接拼列名，生成 PRIMARY KEY (id 主键;号) 而报语法错误，导致特殊列名表无法迁移
func TestMysqlGenTableDDL_SpecialPkColumnName(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "id 主键;号", DataType: "int", IsPrimaryKey: true, Nullable: false},
		{ColumnName: "c_中文 列", DataType: "text", Nullable: true},
	}

	sqls := gen.GenTableDDL(dbi.Table{TableName: "t 特殊;名"}, columns, false)
	require.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "CREATE TABLE `t 特殊;名`")
	assert.Contains(t, sqls[0], "`id 主键;号` int NOT NULL")
	assert.Contains(t, sqls[0], "`c_中文 列` text")
	assert.Contains(t, sqls[0], "PRIMARY KEY (`id 主键;号`)")
}

// TestMysqlGenIndexDDL_SpecialColumnNames 索引列名逐个按标识符引用：
// Quotes内部使用的Quote会按空格切分，含空格的真实列名会被切成两段生成非法DDL
func TestMysqlGenIndexDDL_SpecialColumnNames(t *testing.T) {
	gen := newTestSQLGenerator()
	indexs := []dbi.Index{{IndexName: "idx;名 --x", ColumnName: "c_中文 列,c_分;号"}}

	sqls := gen.GenIndexDDL(dbi.Table{TableName: "t 特殊"}, indexs)
	require.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "ALTER TABLE `t 特殊`")
	assert.Contains(t, sqls[0], "`idx;名 --x`")
	assert.Contains(t, sqls[0], "(`c_中文 列`,`c_分;号`)")
}

// TestMysqlGenTableDDL_BigColDefaultAsExpression BLOB/TEXT/JSON列的默认值必须输出为
// MySQL 8.0.13+支持的表达式默认值形态（DEFAULT ('x')）：MySQL不允许大字列使用字面量默认值
// （报Error 1101），跨库迁移（如sqlite/pg的text默认值列）会直接建表失败；
// varchar等允许字面量的类型必须保持字面量形态，不得多余包裹括号
func TestMysqlGenTableDDL_BigColDefaultAsExpression(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true, Nullable: false},
		{ColumnName: "t", DataType: "text", Nullable: false, ColumnDefault: "abc"},
		{ColumnName: "lt", DataType: "longtext", Nullable: true, ColumnDefault: "it's"},
		{ColumnName: "j", DataType: "json", Nullable: true, ColumnDefault: `{"k":1}`},
		{ColumnName: "b", DataType: "blob", Nullable: true, ColumnDefault: "bin"},
		{ColumnName: "v", DataType: "varchar", CharMaxLength: 32, Nullable: false, ColumnDefault: "abc"},
	}

	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	require.Len(t, sqls, 1)
	ddl := sqls[0]
	assert.Contains(t, ddl, "`t` text NOT NULL DEFAULT ('abc')")
	assert.Contains(t, ddl, "`lt` longtext DEFAULT ('it''s')")
	assert.Contains(t, ddl, "`j` json DEFAULT ('{\"k\":1}')")
	assert.Contains(t, ddl, "`b` blob DEFAULT ('bin')")
	assert.Contains(t, ddl, "`v` varchar(32) NOT NULL DEFAULT 'abc'")
}

// TestMysqlGenTableDDL_NoLiteralDefaultWithoutValue 大字列无默认值时不得凭空生成DEFAULT子句
func TestMysqlGenTableDDL_NoLiteralDefaultWithoutValue(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true, Nullable: false},
		{ColumnName: "t", DataType: "text", Nullable: true},
	}
	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	require.Len(t, sqls, 1)
	assert.NotContains(t, sqls[0], "DEFAULT")
}

// TestMysqlTimeDefaultSql MySQL对「当前日期/时间」默认值的严格语法约束回归：
// 自动初始化子只能是CURRENT_TIMESTAMP且其小数秒参数必须与列fsp严格一致（不匹配即Error 1067），
// CURRENT_DATE/CURRENT_TIME/curdate()/SYSDATE等写法必须改写成8.0.13+的表达式默认值形态
func TestMysqlTimeDefaultSql(t *testing.T) {
	kases := []struct {
		columnType string
		rawDefault string
		expected   string
		handled    bool
	}{
		{"datetime", "CURRENT_TIMESTAMP", " DEFAULT CURRENT_TIMESTAMP", true},
		{"datetime(3)", "CURRENT_TIMESTAMP", " DEFAULT CURRENT_TIMESTAMP(3)", true},
		{"datetime(6)", "CURRENT_TIMESTAMP", " DEFAULT CURRENT_TIMESTAMP(6)", true},
		// 源书写的小数秒与列不一致时以列精度为准（MySQL本身禁止这种不匹配）
		{"datetime(3)", "CURRENT_TIMESTAMP(6)", " DEFAULT CURRENT_TIMESTAMP(3)", true},
		{"timestamp(6)", "now()", " DEFAULT CURRENT_TIMESTAMP(6)", true},
		{"timestamp", "SYSDATE", " DEFAULT CURRENT_TIMESTAMP", true},
		{"datetime", "(getdate())", " DEFAULT CURRENT_TIMESTAMP", true},
		{"datetime", "current_timestamp", " DEFAULT CURRENT_TIMESTAMP", true},
		{"datetime(7)", "CURRENT_TIMESTAMP", " DEFAULT CURRENT_TIMESTAMP(6)", true},
		{"date", "CURRENT_TIMESTAMP", " DEFAULT (CURRENT_DATE)", true},
		{"date", "curdate()", " DEFAULT (CURRENT_DATE)", true},
		{"date", "(CURRENT_DATE)", " DEFAULT (CURRENT_DATE)", true},
		{"time", "CURRENT_TIME", " DEFAULT (CURRENT_TIME)", true},
		{"time(2)", "CURTIME()", " DEFAULT (CURRENT_TIME)", true},
		// 字符串列的默认值内容可能就是这个文本，不得改写成SQL关键字
		{"varchar(32)", "CURRENT_TIMESTAMP", "", false},
		{"datetime", "'2020-01-01 00:00:00'", "", false},
		{"datetime", "NULL", "", false},
		{"datetime", "", "", false},
		{"int", "3", "", false},
	}

	for _, k := range kases {
		defVal, handled := mysqlTimeDefaultSql(k.rawDefault, k.columnType)
		assert.Equal(t, k.handled, handled, "raw=%q type=%q", k.rawDefault, k.columnType)
		assert.Equal(t, k.expected, defVal, "raw=%q type=%q", k.rawDefault, k.columnType)
	}
}

// TestMysqlGenTableDDL_TimeDefault 端到端校验建表DDL的时间默认值与列fsp一致，
// 并确认字符串列的同形关键字默认值仍按字面量引用
func TestMysqlGenTableDDL_TimeDefault(t *testing.T) {
	gen := newTestSQLGenerator()
	columns := []dbi.Column{
		{ColumnName: "id", DataType: "int", IsPrimaryKey: true, Nullable: false},
		{ColumnName: "c0", DataType: "datetime", Nullable: false, ColumnDefault: "CURRENT_TIMESTAMP"},
		{ColumnName: "c3", DataType: "datetime", NumPrecision: 3, Nullable: false, ColumnDefault: "CURRENT_TIMESTAMP"},
		{ColumnName: "c6", DataType: "timestamp", NumPrecision: 6, Nullable: true, ColumnDefault: "now()"},
		{ColumnName: "dt", DataType: "date", Nullable: true, ColumnDefault: "SYSDATE"},
		{ColumnName: "tm", DataType: "time", NumPrecision: 2, Nullable: true, ColumnDefault: "CURRENT_TIMESTAMP"},
		{ColumnName: "s", DataType: "varchar", CharMaxLength: 32, Nullable: true, ColumnDefault: "CURRENT_TIMESTAMP"},
	}

	sqls := gen.GenTableDDL(dbi.Table{TableName: "t1"}, columns, false)
	require.Len(t, sqls, 1)
	ddl := sqls[0]
	assert.Contains(t, ddl, "`c0` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP")
	assert.Contains(t, ddl, "`c3` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)")
	assert.Contains(t, ddl, "`c6` timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6)")
	assert.Contains(t, ddl, "`dt` date DEFAULT (CURRENT_DATE)")
	assert.Contains(t, ddl, "`tm` time(2) DEFAULT (CURRENT_TIME)")
	assert.Contains(t, ddl, "`s` varchar(32) DEFAULT 'CURRENT_TIMESTAMP'")
}
