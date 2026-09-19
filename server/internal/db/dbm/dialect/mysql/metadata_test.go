package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestColumnDefaultOf 列默认值元数据归一的表驱动测试。
//
// 依据（均在本地mysql:8.0.46真实库实测确认）：
//   - information_schema.COLUMNS.COLUMN_DEFAULT：无默认值为SQL NULL；DEFAULT ”为空串；
//     DEFAULT NULL为SQL NULL；字符串默认值呈现为去引号未转义的裸原始值。
//   - NOT NULL列声明DEFAULT NULL建表即报ERROR 1067，故非空列上的裸NULL必为字符串'NULL'。
func TestColumnDefaultOf(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		nullable bool
		want     string
	}{
		{"无默认值-SQL NULL", nil, true, ""},
		{"无默认值-SQL NULL-非空列", nil, false, ""},
		{"空串默认值归一为字面量形态", "", false, "''"},
		{"空串默认值-可空列", "", true, "''"},
		{"mysql8.0裸文本原样呈现", "abc", true, "abc"},
		{"含特殊字符的裸文本不被加工", "it's", false, "it's"},
		{"含括号字面量保持裸值交由方言判定", "(0)", true, "(0)"},
		{"可空列裸NULL保留无默认值语义-兼容5.7", "NULL", true, "NULL"},
		{"非空列裸NULL还原为字符串字面量", "NULL", false, "'NULL'"},
		{"数值负默认值", "-1", false, "-1"},
		{"表达式默认值8.0.13+", "(uuid())", true, "(uuid())"},
		{"[]byte非空值", []byte("x"), false, "x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, columnDefaultOf(tt.val, tt.nullable))
		})
	}
}

// TestColumnDefaultOf_NilByteSlice 扫描出的空[]byte（非nil切片但长度为0）不能panic，
// 且需与SQL NULL区分：长度为0的空切片代表空串默认值。
func TestColumnDefaultOf_NilByteSlice(t *testing.T) {
	var nilSlice []byte
	assert.Equal(t, "", columnDefaultOf(nilSlice, false), "nil切片等同SQL NULL")
	assert.Equal(t, "''", columnDefaultOf([]byte{}, false), "空切片为空串默认值")
}

// gcCreateSql 真实mysql:8.0.46的SHOW CREATE TABLE输出（反引号列名含反斜杠、表达式内含
// 单引号/反斜杠/非ASCII字面量、VIRTUAL列带NOT NULL、以及普通列/索引/约束行）
const gcCreateSql = "CREATE TABLE `t_gexpr2` (\n" +
	"  `id` int NOT NULL,\n" +
	"  `a` int NOT NULL,\n" +
	"  `b\\b` int NOT NULL,\n" +
	"  `g_cn` varchar(200) GENERATED ALWAYS AS (concat(`a`,_utf8mb4'中文',_utf8mb4'A-b')) VIRTUAL NOT NULL,\n" +
	"  `g_q` varchar(200) GENERATED ALWAYS AS (concat(`a`,_utf8mb4'-',_utf8mb4'it\\'s',_utf8mb4'\\\\')) STORED,\n" +
	"  `g_bs` varchar(200) GENERATED ALWAYS AS (concat(`b\\b`,_utf8mb4'-x')) STORED,\n" +
	"  `g_key` int GENERATED ALWAYS AS ((`a` * 2)) STORED,\n" +
	"  `g_paren` varchar(20) GENERATED ALWAYS AS (concat(`a`,'(',1)) VIRTUAL,\n" +
	"  PRIMARY KEY (`id`),\n" +
	"  KEY `idx_g_cn` (`g_cn`)\n" +
	") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"

// TestParseMysqlGeneratedColumns 生成列定义原文必须逐字节取回（可直接重放入DDL），
// 且不得把普通列/索引行误判为生成列
func TestParseMysqlGeneratedColumns(t *testing.T) {
	genColumns := parseMysqlGeneratedColumns(gcCreateSql)

	// 只识别出生成列，普通列名不入表
	assert.Len(t, genColumns, 5)
	for _, plain := range []string{"id", "a", "b\\b", "idx_g_cn", "PRIMARY"} {
		_, ok := genColumns[strings.ToLower(plain)]
		assert.False(t, ok, "[%s]不是生成列", plain)
	}

	assert.Equal(t, "concat(`a`,_utf8mb4'中文',_utf8mb4'A-b')", genColumns["g_cn"].expr)
	assert.False(t, genColumns["g_cn"].stored, "VIRTUAL列不得标为存储")
	assert.Equal(t, "concat(`a`,_utf8mb4'-',_utf8mb4'it\\'s',_utf8mb4'\\\\')", genColumns["g_q"].expr)
	assert.True(t, genColumns["g_q"].stored, "STORED列必须标为存储")
	// 表达式内的标识符含反斜杠：不得被当作转义序列加工
	assert.Equal(t, "concat(`b\\b`,_utf8mb4'-x')", genColumns["g_bs"].expr)
	// 外层多余一层括号属于表达式原文（DDL再包一层即MySQL自己的形态）
	assert.Equal(t, "(`a` * 2)", genColumns["g_key"].expr)
	// 字面量内的括号不得当作闭合点
	assert.Equal(t, "concat(`a`,'(',1)", genColumns["g_paren"].expr)
}

// TestParseMysqlGeneratedColumnsAbnormal 形态未知时必须不识别（退回普通列+插源值），宁可保守不可猜
func TestParseMysqlGeneratedColumnsAbnormal(t *testing.T) {
	cases := map[string]string{
		// 字面量未闭合（表达式内含裸换行使行被截断）
		"unclosed literal": "CREATE TABLE `t` (\n  `g` int GENERATED ALWAYS AS (concat('a\n) VIRTUAL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB",
		"无存储关键字":           "CREATE TABLE `t` (\n  `g` int GENERATED ALWAYS AS ((`a` + 1)),\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB",
		"关键字前置":            "CREATE TABLE `t` (\n  `g` int GENERATED ALWAYS AS ((`a` + 1)) NOT NULL VIRTUAL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB",
		"空表达式":             "CREATE TABLE `t` (\n  `g` int GENERATED ALWAYS AS () VIRTUAL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB",
	}
	for name, createSql := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Empty(t, parseMysqlGeneratedColumns(createSql))
		})
	}
}

// TestParseMysqlGeneratedColumnsMariaDB MariaDB的形态差异：PERSISTENT等价STORED，且类型带显示宽度
func TestParseMysqlGeneratedColumnsMariaDB(t *testing.T) {
	createSql := "CREATE TABLE `t_m` (\n  `id` int(11) NOT NULL,\n  `a` int(11) NOT NULL,\n  `g` int(11) GENERATED ALWAYS AS (`a` + 1) PERSISTENT,\n  `v` int(11) GENERATED ALWAYS AS (`a` * 2) VIRTUAL COMMENT '翻倍',\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB"
	genColumns := parseMysqlGeneratedColumns(createSql)
	require.Len(t, genColumns, 2)
	assert.Equal(t, "`a` + 1", genColumns["g"].expr)
	assert.True(t, genColumns["g"].stored, "PERSISTENT等价STORED")
	assert.Equal(t, "`a` * 2", genColumns["v"].expr)
	assert.False(t, genColumns["v"].stored)
}

// TestSplitMysqlParenContent 括号切分需尊重字面量与标识符边界（企业级切割的底线）
func TestSplitMysqlParenContent(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantBody string
		wantTail string
		wantOk   bool
	}{
		{"基本", "(a)", "a", "", true},
		{"嵌套", "((a + 1))", "(a + 1)", "", true},
		{"尾部内容", "(`a` * 2) STORED", "`a` * 2", " STORED", true},
		{"字面量含括号", "(concat('(', ')'))", "concat('(', ')')", "", true},
		{"字面量含转义引号", `(concat('it\'s(', 1)) VIRTUAL)`, `concat('it\'s(', 1)`, " VIRTUAL)", true},
		{"反引号标识符含括号", "(concat(`a(b`, 1))", "concat(`a(b`, 1)", "", true},
		{"双引号字面量", "(\"(\")", "\"(\"", "", true},
		{"单引号双写", "(concat('a''(', 1))", "concat('a''(', 1)", "", true},
		{"字面量未闭合", "(concat('a, 1)", "", "", false},
		{"括号未闭合", "((a + 1)", "", "", false},
		{"非括号开头", "a + 1", "", "", false},
		{"空内容", "()", "", "", true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, tail, ok := splitMysqlParenContent(tt.input)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.wantBody, body)
				assert.Equal(t, tt.wantTail, tail)
			}
		})
	}
}

// TestMysqlShowCreateText 结果集列名大小写差异下均需取到建表文本，且不得误取其他列
func TestMysqlShowCreateText(t *testing.T) {
	assert.Equal(t, "CREATE TABLE `t`", mysqlShowCreateText(map[string]any{"Table": "t", "Create Table": "CREATE TABLE `t`"}))
	assert.Equal(t, "CREATE TABLE `t`", mysqlShowCreateText(map[string]any{"create table": "CREATE TABLE `t`"}))
	assert.Equal(t, "", mysqlShowCreateText(map[string]any{"Table": "t"}), "无建表列时返回空串而非误取表名")
}
