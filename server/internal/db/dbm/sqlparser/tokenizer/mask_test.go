package tokenizer

import (
	"strings"
	"testing"
)

// TestMaskSqlComments 注释掩码：普通注释等长替换为空白，字面量保留原文，可执行注释解壳保留内容。
// 供导入侧「按语句整体文本识别事务控制语句」使用（切割保留注释后不掩码会误判）
func TestMaskSqlComments(t *testing.T) {
	cases := []struct {
		name string
		cfg  DialectConfig
		sql  string
		want string
	}{
		{"行注释掩码且换行保留", StdConfig, "-- c\nBEGIN", "    \nBEGIN"},
		{"块注释掩码", StdConfig, "/* b */ COMMIT;", "        COMMIT;"},
		{"分段注释头后的BEGIN可识别", MysqlConfig, "-- ----\n-- Data: t \n-- ----\nBEGIN;", "       \n           \n       \nBEGIN;"},
		{"mysql 井号行注释", MysqlConfig, "# c\nCOMMIT;", "   \nCOMMIT;"},
		{"mysql 可执行注释解壳保留内容", MysqlConfig, "/*!40101 SET autocommit=0 */", "         SET autocommit=0   "},
		{"mssql 井号是临时表前缀不掩码", MssqlConfig, "SELECT * FROM #tmp -- c", "SELECT * FROM #tmp     "},
		{"clickhouse 井号为行注释", ClickhouseConfig, "SELECT 1 # c\nBEGIN", "SELECT 1    \nBEGIN"},
		{"字符串内的注释符属于数据", MysqlConfig, "SELECT '-- not comment', /*x*/ 1", "SELECT '-- not comment',       1"},
		{"引用标识符内的注释符属于数据", MssqlConfig, "SELECT [a--b] /* c */ 1", "SELECT [a--b]         1"},
		{"pg 嵌套块注释整体掩码", PgConfig, "SELECT /* a /* b */ c */ 1", "SELECT                   1"},
		{"pg dollar-quote 体保留原文", PgConfig, "SELECT $$ -- not comment $$ ;", "SELECT $$ -- not comment $$ ;"},
		{"oracle q-quote 体保留原文", OracleConfig, "SELECT q'[--x]' FROM dual", "SELECT q'[--x]' FROM dual"},
		{"标准 SQL 双横线无需后随空白即为注释", StdConfig, "SELECT 1--2", "SELECT 1   "},
		{"mysql 双横线无后随空白为减法不掩码", MysqlConfig, "SELECT 1--2", "SELECT 1--2"},
		{"未闭合块注释掩码到文末", StdConfig, "SELECT 1; /* open\nSELECT 2", "SELECT 1;        \n        "},
		{"无注释文本原样返回", StdConfig, "SELECT 1", "SELECT 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MaskSqlComments(c.sql, c.cfg)
			if got != c.want {
				t.Fatalf("掩码不符:\n got=%q\nwant=%q", got, c.want)
			}
			// 不变量：等长、换行数不变（掩码后仍可按原下标回查）
			if len(got) != len(c.sql) {
				t.Fatalf("长度改变: got=%d want=%d", len(got), len(c.sql))
			}
			if strings.Count(got, "\n") != strings.Count(c.sql, "\n") {
				t.Fatal("换行数改变，行号语义将与原文错位")
			}
			for i := range c.sql {
				if c.sql[i] == '\n' && got[i] != '\n' {
					t.Fatalf("第 %d 字节换行被改写", i)
				}
			}
		})
	}
}

// TestMaskSqlCommentsIdempotent 掩码结果再次掩码必须不变（幂等），否则多次判定会漂移
func TestMaskSqlCommentsIdempotent(t *testing.T) {
	texts := []string{
		"-- c\nBEGIN;",
		"/* a /* b */ c */ SELECT 1;",
		"SELECT 'a/*x*/b', \"c--d\" FROM `t`; -- e",
		"/*!40101 SET NAMES utf8 */;",
		"SELECT q'[--]' FROM dual; # x",
	}
	for _, d := range []struct {
		name string
		cfg  DialectConfig
	}{{"std", StdConfig}, {"mysql", MysqlConfig}, {"pg", PgConfig}, {"oracle", OracleConfig}, {"mssql", MssqlConfig}, {"clickhouse", ClickhouseConfig}, {"sqlite", SqliteConfig}, {"dm", DmConfig}} {
		for _, text := range texts {
			once := MaskSqlComments(text, d.cfg)
			if twice := MaskSqlComments(once, d.cfg); twice != once {
				t.Fatalf("[%s] 掩码不幂等:\n first=%q\nsecond=%q", d.name, once, twice)
			}
		}
	}
}
