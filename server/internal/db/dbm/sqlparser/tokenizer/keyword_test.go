package tokenizer

import (
	"testing"
)

// TestLeadingKeyword 首关键字提取：跳过前导空白与注释区域，注释原文不参与类型判定
func TestLeadingKeyword(t *testing.T) {
	cases := []struct {
		name string
		cfg  DialectConfig
		sql  string
		want string
	}{
		{"无前导噪声", StdConfig, "SELECT 1", "select"},
		{"前导空白", StdConfig, "\n\t  update t set a = 1", "update"},
		{"块注释前缀", StdConfig, "/* header */ select 1", "select"},
		{"行注释前缀", StdConfig, "-- 说明\nselect 1", "select"},
		{"行注释含关键字干扰", StdConfig, "-- select 的历史\nDELETE FROM t", "delete"},
		{"嵌套块注释", PgConfig, "/* a /* b */ c */ select 1", "select"},
		{"mysql # 注释前缀", MysqlConfig, "# 备注\nselect 1", "select"},
		{"标准SQL的#非注释", StdConfig, "# op\nselect 1", ""},
		{"mysql可执行注释", MysqlConfig, "/*!40101 SET NAMES utf8 */", "set"},
		{"普通注释内的可执行样式文本", StdConfig, "/*+ hint */ select 1", "select"},
		{"dollar-quote前无关键字", PgConfig, " $$ x $$ ", ""},
		{"纯注释", StdConfig, "-- only\n/* block */", ""},
		{"空串", StdConfig, "", ""},
		{"引用标识符开头", MssqlConfig, `[t]`, ""},
		{"未闭合块注释", StdConfig, "/* unclosed select", ""},
		{"大小写混合", StdConfig, "DeLeTe FROM t", "delete"},
		{"关键字含下划线", StdConfig, "insert_all FROM t", "insert_all"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := LeadingKeyword(c.sql, c.cfg); got != c.want {
				t.Fatalf("LeadingKeyword(%q)=%q, want %q", c.sql, got, c.want)
			}
		})
	}
}
