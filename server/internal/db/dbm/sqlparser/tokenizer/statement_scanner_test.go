package tokenizer

import (
	"errors"
	"strings"
	"testing"
)

// splitByCfg 使用指定方言能力表切割，返回语句列表（注释原文保留）
func splitByCfg(t *testing.T, sql string, cfg DialectConfig) []string {
	t.Helper()
	stmts, err := SplitStatements(sql, cfg, ';')
	if err != nil {
		t.Fatalf("切割失败: %v", err)
	}
	return stmts
}

// TestScannerDialectMatrix 各方言能力位下的复杂 SQL 切割矩阵
func TestScannerDialectMatrix(t *testing.T) {
	cases := []struct {
		name string
		cfg  DialectConfig
		sql  string
		want []string
	}{
		// ---- mysql ----
		{"mysql 双横线无空白为减法", MysqlConfig, "SELECT 1--2;SELECT 2;", []string{"SELECT 1--2", "SELECT 2"}},
		{"mysql 双横线有空白为注释", MysqlConfig, "SELECT 1 -- c\n;SELECT 2;", []string{"SELECT 1 -- c", "SELECT 2"}},
		{"mysql # 行注释", MysqlConfig, "SELECT 1 # c\n;SELECT 2;", []string{"SELECT 1 # c", "SELECT 2"}},
		{"mysql 反斜杠转义", MysqlConfig, `SELECT 'a\'b';SELECT 2;`, []string{`SELECT 'a\'b'`, "SELECT 2"}},
		{"mysql 双引号字符串内含反斜杠", MysqlConfig, `SELECT "a\"b";SELECT 2;`, []string{`SELECT "a\"b"`, "SELECT 2"}},
		{"mysql 反引号标识符含分号", MysqlConfig, "SELECT `a;b` FROM `t`;SELECT 2;", []string{"SELECT `a;b` FROM `t`", "SELECT 2"}},
		{"mysql 反引号双写转义", MysqlConfig, "SELECT `a``b`;SELECT 2;", []string{"SELECT `a``b`", "SELECT 2"}},
		{"mysql 注释粘连不破坏 token", MysqlConfig, "SELECT count(*)/*x*/FROM t;", []string{"SELECT count(*)/*x*/FROM t"}},
		{"mysql 存储过程整体一条", MysqlConfig, "CREATE PROCEDURE p() BEGIN SELECT 1; SELECT 2; END;", []string{"CREATE PROCEDURE p() BEGIN SELECT 1; SELECT 2; END"}},
		{"mysql loop 作表名不误入块", MysqlConfig, "SELECT * FROM loop;SELECT 2;", []string{"SELECT * FROM loop", "SELECT 2"}},
		{"mysql IF 函数不入块", MysqlConfig, "SELECT IF(a, 1, 2) FROM t;SELECT 2;", []string{"SELECT IF(a, 1, 2) FROM t", "SELECT 2"}},
		{"mysql 过程体内 END IF", MysqlConfig, "CREATE PROCEDURE p() BEGIN IF a THEN SELECT 1; END IF; SELECT 2; END;", []string{"CREATE PROCEDURE p() BEGIN IF a THEN SELECT 1; END IF; SELECT 2; END"}},
		{"mysql 过程体内 LOOP", MysqlConfig, "CREATE PROCEDURE p() BEGIN LOOP SET a=1; END LOOP; END;", []string{"CREATE PROCEDURE p() BEGIN LOOP SET a=1; END LOOP; END"}},
		// ---- 标准 SQL / pg ----
		{"标准 SQL 反斜杠为普通字符", StdConfig, `SELECT '\';SELECT 2;`, []string{`SELECT '\'`, "SELECT 2"}},
		{"pg 反斜杠为普通字符", PgConfig, `SELECT 'a\b';SELECT 2;`, []string{`SELECT 'a\b'`, "SELECT 2"}},
		{"pg E 串反斜杠转义", PgConfig, `SELECT E'a\'b';SELECT 2;`, []string{`SELECT E'a\'b'`, "SELECT 2"}},
		{"pg 列名尾字母 e 不开启 E 串", PgConfig, `SELECT e'a\b' FROM t;`, []string{`SELECT e'a\b' FROM t`}},
		{"pg 双写引号", PgConfig, `SELECT 'a''b;c';SELECT 2;`, []string{`SELECT 'a''b;c'`, "SELECT 2"}},
		{"pg dollar quote 函数体", PgConfig, "CREATE FUNCTION f() RETURNS int AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql;SELECT 2;", []string{"CREATE FUNCTION f() RETURNS int AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql", "SELECT 2"}},
		{"pg 带标签 dollar quote", PgConfig, "SELECT $tag$a;b$tag$;SELECT 2;", []string{"SELECT $tag$a;b$tag$", "SELECT 2"}},
		{"pg 嵌套块注释", PgConfig, "SELECT /* a /* b */ c */ 1;", []string{"SELECT /* a /* b */ c */ 1"}},
		{"pg 事务 BEGIN 正常切割", PgConfig, "BEGIN;SELECT 1;COMMIT;", []string{"BEGIN", "SELECT 1", "COMMIT"}},
		{"pg BEGIN TRANSACTION 正常切割", PgConfig, "BEGIN TRANSACTION;SELECT 1;", []string{"BEGIN TRANSACTION", "SELECT 1"}},
		{"pg DO 匿名块", PgConfig, "DO $$ BEGIN RAISE NOTICE 'x'; END $$;", []string{"DO $$ BEGIN RAISE NOTICE 'x'; END $$"}},
		{"非嵌套方言块注释提前闭合", MysqlConfig, "SELECT /* a /* b */ 1;", []string{"SELECT /* a /* b */ 1"}},
		// ---- clickhouse ----
		{"clickhouse 双横线无需空白即注释", ClickhouseConfig, "SELECT 1 --2\n;SELECT 2;", []string{"SELECT 1 --2", "SELECT 2"}},
		{"clickhouse 反引号", ClickhouseConfig, "SELECT `a;b`;SELECT 2;", []string{"SELECT `a;b`", "SELECT 2"}},
		// ---- mssql ----
		{"mssql 方括号标识符含分号", MssqlConfig, "SELECT [a;b] FROM [t];SELECT 2;", []string{"SELECT [a;b] FROM [t]", "SELECT 2"}},
		{"mssql ]] 为转义右括号", MssqlConfig, "SELECT [a]]];SELECT 2;", []string{"SELECT [a]]]", "SELECT 2"}},
		{"mssql # 为临时表前缀非注释", MssqlConfig, "SELECT * FROM #tmp;SELECT 2;", []string{"SELECT * FROM #tmp", "SELECT 2"}},
		{"mssql BEGIN TRY..END TRY", MssqlConfig, "BEGIN TRY SELECT 1; END TRY BEGIN CATCH SELECT 2; END CATCH;", []string{"BEGIN TRY SELECT 1; END TRY BEGIN CATCH SELECT 2; END CATCH"}},
		{"mssql 裸 BEGIN 事务", MssqlConfig, "BEGIN TRANSACTION;SELECT 1;COMMIT TRANSACTION;", []string{"BEGIN TRANSACTION", "SELECT 1", "COMMIT TRANSACTION"}},
		// ---- oracle ----
		{"oracle q quote 含分号", OracleConfig, "SELECT q'[a;b]' FROM dual;SELECT 2 FROM dual;", []string{"SELECT q'[a;b]' FROM dual", "SELECT 2 FROM dual"}},
		{"oracle q quote 含单引号", OracleConfig, `SELECT q'[';']' FROM dual;SELECT 2 FROM dual;`, []string{`SELECT q'[';']' FROM dual`, "SELECT 2 FROM dual"}},
		{"oracle nq/uq 前缀", OracleConfig, "SELECT nq'[a;b]', uq'{c;d}' FROM dual;", []string{"SELECT nq'[a;b]', uq'{c;d}' FROM dual"}},
		{"oracle 匿名块一条", OracleConfig, "DECLARE x INT; BEGIN x:=1; END;", []string{"DECLARE x INT; BEGIN x:=1; END"}},
		{"oracle END LOOP 不出块", OracleConfig, "BEGIN FOR i IN 1..3 LOOP x:=1; END LOOP; END;", []string{"BEGIN FOR i IN 1..3 LOOP x:=1; END LOOP; END"}},
		{"oracle 保留 hint 注释", OracleConfig, "SELECT /*+ INDEX(t idx) */ a FROM t;", []string{"SELECT /*+ INDEX(t idx) */ a FROM t"}},
		// ---- sqlite ----
		{"sqlite 反引号与方括号", SqliteConfig, "SELECT `a;b`, [c;d];SELECT 2;", []string{"SELECT `a;b`, [c;d]", "SELECT 2"}},
		{"sqlite 反斜杠为普通字符", SqliteConfig, `SELECT '\';SELECT 2;`, []string{`SELECT '\'`, "SELECT 2"}},
		// ---- 复杂字符串与文件形态 ----
		{"流起始 UTF-8 BOM 剥离", StdConfig, "\ufeffSELECT 1;\nSELECT 2;", []string{"SELECT 1", "SELECT 2"}},
		{"相邻字符串字面量跨行拼接", PgConfig, "SELECT 'a'\n'b;c';", []string{"SELECT 'a'\n'b;c'"}},
		{"字符串含 NUL 字节", StdConfig, "SELECT 'a\x00b;c';SELECT 2;", []string{"SELECT 'a\x00b;c'", "SELECT 2"}},
		{"mssql N 前缀字符串", MssqlConfig, "SELECT N'a;b';SELECT 2;", []string{"SELECT N'a;b'", "SELECT 2"}},
		{"pg 位置参数 $1 不开 dollar-quote", PgConfig, "SELECT * FROM t WHERE id=$1 AND n=$2;SELECT 2;", []string{"SELECT * FROM t WHERE id=$1 AND n=$2", "SELECT 2"}},
		{"pg 大写 tag 的 dollar-quote", PgConfig, "SELECT $FN$a;b$FN$;SELECT 2;", []string{"SELECT $FN$a;b$FN$", "SELECT 2"}},
		{"oracle q-quote 圆括号镜像闭合", OracleConfig, "SELECT q'(a)b)' FROM dual;SELECT 2 FROM dual;", []string{"SELECT q'(a)b)' FROM dual", "SELECT 2 FROM dual"}},
		{"oracle q-quote 尖括号定界符", OracleConfig, "SELECT q'<a;b>' FROM dual;", []string{"SELECT q'<a;b>' FROM dual"}},
		// ---- 复合块与过程脚本 ----
		{"oracle 嵌套 CASE", OracleConfig, "SELECT CASE WHEN CASE WHEN x THEN 1 END THEN 2 END FROM dual;", []string{"SELECT CASE WHEN CASE WHEN x THEN 1 END THEN 2 END FROM dual"}},
		{"mysql 嵌套 CASE", MysqlConfig, "SELECT CASE WHEN CASE WHEN x THEN 1 END THEN 2 END;SELECT 2;", []string{"SELECT CASE WHEN CASE WHEN x THEN 1 END THEN 2 END", "SELECT 2"}},
		{"oracle 匿名块 EXCEPTION 段", OracleConfig, "BEGIN x:=1; EXCEPTION WHEN OTHERS THEN NULL; END;", []string{"BEGIN x:=1; EXCEPTION WHEN OTHERS THEN NULL; END"}},
		{"mysql 块内字符串含 END 文本不出块", MysqlConfig, "CREATE PROCEDURE p() BEGIN SELECT 'END;'; END;SELECT 2;", []string{"CREATE PROCEDURE p() BEGIN SELECT 'END;'; END", "SELECT 2"}},
		{"mysql 注释中的 END 不出块（无配对 END 则关闭块感知重切）", MysqlConfig, "CREATE PROCEDURE p() BEGIN SELECT 1; -- END\nSELECT 2;", []string{"CREATE PROCEDURE p() BEGIN SELECT 1", "-- END\nSELECT 2"}},
		{"CASE 缺配对 END 不吞并后续语句", MysqlConfig, "SELECT CASE WHEN a THEN 1;SELECT 2;", []string{"SELECT CASE WHEN a THEN 1", "SELECT 2"}},
		{"BEGIN 无分号且块未闭合时降级重切", PgConfig, "BEGIN\nSELECT 1;SELECT 2;SELECT 3;", []string{"BEGIN\nSELECT 1", "SELECT 2", "SELECT 3"}},
		{"mysql 触发器体", MysqlConfig, "CREATE TRIGGER tr BEFORE INSERT ON t FOR EACH ROW BEGIN SET @a=1; SET @b=2; END;\nSELECT 2;", []string{"CREATE TRIGGER tr BEFORE INSERT ON t FOR EACH ROW BEGIN SET @a=1; SET @b=2; END", "SELECT 2"}},
		{"mysql IF..ELSE..END IF 完整过程体", MysqlConfig, "CREATE PROCEDURE p() BEGIN IF a THEN SELECT 1; ELSE SELECT 2; END IF; END;", []string{"CREATE PROCEDURE p() BEGIN IF a THEN SELECT 1; ELSE SELECT 2; END IF; END"}},
		{"mssql WHILE..BEGIN..END（DECLARE 独立成句）", MssqlConfig, "DECLARE @i INT; WHILE @i < 10 BEGIN SET @i = @i + 1; END;\nSELECT 4;", []string{"DECLARE @i INT; WHILE @i < 10 BEGIN SET @i = @i + 1; END", "SELECT 4"}},
		{"mssql IF..BEGIN..END ELSE..BEGIN..END", MssqlConfig, "IF 1=1 BEGIN SELECT 1; END ELSE BEGIN SELECT 2; END;", []string{"IF 1=1 BEGIN SELECT 1; END ELSE BEGIN SELECT 2; END"}},
		// ---- 真实查询形态 ----
		{"mysql CTE + 窗口函数", MysqlConfig, "WITH t AS (SELECT ROW_NUMBER() OVER (PARTITION BY a ORDER BY b) rn FROM x) SELECT * FROM t WHERE rn=1;SELECT 2;", []string{"WITH t AS (SELECT ROW_NUMBER() OVER (PARTITION BY a ORDER BY b) rn FROM x) SELECT * FROM t WHERE rn=1", "SELECT 2"}},
		{"mysql SET 用户变量含子查询", MysqlConfig, "SET @x:=(SELECT 1);SELECT @x;", []string{"SET @x:=(SELECT 1)", "SELECT @x"}},
		{"mysql CRLF 下的行注释", MysqlConfig, "SELECT 1; -- c\r\nSELECT 2;\r\n", []string{"SELECT 1", "-- c\r\nSELECT 2"}},
		// 行注释仅以 \n 终止（与 MySQL 服务端/psql 一致）：若全文只用 CR 换行，注释后的文本归入注释而不产出语句
		{"仅 CR 换行的行注释延伸至文末", MysqlConfig, "SELECT 1; -- c\rSELECT 2;", []string{"SELECT 1"}},
		// 客户端指令型分隔符不在能力表范围内（见 DialectSplitter 已知限制），此用例锁定当前行为不得恶化为吞并整段脚本
		{"mysql DELIMITER 指令不识别（已知限制）", MysqlConfig, "DELIMITER ;;\nCREATE PROCEDURE p() BEGIN SELECT 1;;\nEND;;\nDELIMITER ;", []string{"DELIMITER", "CREATE PROCEDURE p() BEGIN SELECT 1;;\nEND", "DELIMITER"}},
		// ---- 尾部无分隔符 / 空语句 ----
		{"尾部无分隔符", StdConfig, "SELECT 1;SELECT 2", []string{"SELECT 1", "SELECT 2"}},
		{"连续分隔符不产出空语句", StdConfig, "SELECT 1;;SELECT 2;;;", []string{"SELECT 1", "SELECT 2"}},
		{"纯注释不产出语句", StdConfig, "-- only comment\n", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := splitByCfg(t, c.sql, c.cfg)
			if len(got) != len(c.want) {
				t.Fatalf("语句数不符: got=%d want=%d got=%q", len(got), len(c.want), got)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("第 %d 条不符:\n got=%q\nwant=%q", i+1, got[i], c.want[i])
				}
			}
			// 切割不得改写语句内容：每条语句必须能在原文中按序定位
			assertSubsequence(t, c.sql, got)
		})
	}
}

// assertSubsequence 语句必须是原文的有序子串（保证注释、空白等未被改写）
func assertSubsequence(t *testing.T, sql string, stmts []string) {
	t.Helper()
	from := 0
	for _, stmt := range stmts {
		idx := strings.Index(sql[from:], stmt)
		if idx < 0 {
			t.Fatalf("语句非原文有序子串: %q", stmt)
		}
		from += idx + len(stmt)
	}
}

// TestScannerUnterminated 未闭合字面量/注释必须报错且带行号（不再静默吞并后续脚本）
func TestScannerUnterminated(t *testing.T) {
	cases := []struct {
		name string
		cfg  DialectConfig
		sql  string
		line int
	}{
		{"单引号未闭合", StdConfig, "SELECT 1;\nSELECT 'abc;\nSELECT 2;", 2},
		{"mysql 反斜杠致引号未闭合", MysqlConfig, `SELECT 'a\';`, 1},
		{"块注释未闭合", StdConfig, "/* open\nSELECT 1;", 1},
		{"pg 嵌套块注释未闭合", PgConfig, "/* a /* b */\nSELECT 1;", 1},
		{"pg dollar quote 未闭合", PgConfig, "SELECT $$abc;", 1},
		{"oracle q quote 未闭合", OracleConfig, "SELECT q'[abc;", 1},
		{"mysql 反引号未闭合", MysqlConfig, "SELECT `abc;", 1},
		{"mssql 方括号未闭合", MssqlConfig, "SELECT [abc;", 1},
		{"mssql 右括号转义后缺结束符", MssqlConfig, "SELECT [a]];SELECT 2;", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := SplitStatements(c.sql, c.cfg, ';')
			var ue *UnterminatedError
			if !errors.As(err, &ue) {
				t.Fatalf("期望未闭合错误, got=%v", err)
			}
			if ue.Line != c.line {
				t.Fatalf("行号不符: got=%d want=%d", ue.Line, c.line)
			}
		})
	}
}

// TestScannerStreaming 流式喂入（逐行/逐字符）结果必须与一次性切割一致
func TestScannerStreaming(t *testing.T) {
	scripts := map[string]DialectConfig{
		"SELECT a-- c\nFROM t;\nSELECT 'x;y';\nCREATE PROCEDURE p() BEGIN SELECT 1; END;\n": MysqlConfig,
		"DO $$ BEGIN EXECUTE 'a;b'; END $$;\nSELECT /* a /* b */ c */ 1;\n":                 PgConfig,
		"SELECT q'[a;b]', [c;d] FROM t;\n":                                                  OracleConfig,
		// 未闭合的块起始词：流式逐块喂入同样需走 EOF 降级重切，结果与一次性切割一致
		"SELECT CASE WHEN a THEN 1;\nSELECT 2;\n": MysqlConfig,
		// BOM 与多字节字符均可能被输入块切断（逐字节喂入时）
		"\ufeffSELECT '中文🙂;x';\nSELECT 2;\n": StdConfig,
	}
	for script, cfg := range scripts {
		want := splitByCfg(t, script, cfg)
		if len(want) == 0 {
			t.Fatalf("用例无输出: %q", script)
		}
		for _, chunkSize := range []int{1, 3, 7} {
			scanner := NewStatementScanner(cfg, ';')
			var got []string
			for i := 0; i < len(script); i += chunkSize {
				end := min(i+chunkSize, len(script))
				stmts, err := scanner.Feed(script[i:end])
				if err != nil {
					t.Fatalf("流式切割失败: %v", err)
				}
				got = append(got, stmts...)
			}
			rest, err := scanner.Finish()
			if err != nil {
				t.Fatalf("流式切割收尾失败: %v", err)
			}
			got = append(got, rest...)
			if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
				t.Fatalf("chunk=%d 流式结果与一次性不一致:\n got=%q\nwant=%q", chunkSize, got, want)
			}
		}
	}
}

// TestScannerPreservesExecutableComments 可执行注释与 hint 必须原样保留（否则导入语义丢失）
func TestScannerPreservesExecutableComments(t *testing.T) {
	got := splitByCfg(t, "/*!40101 SET NAMES utf8 */;\nSELECT /*+ LEADING(t) */ 1;\n", MysqlConfig)
	want := []string{"/*!40101 SET NAMES utf8 */", "SELECT /*+ LEADING(t) */ 1"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("可执行注释未保留: %q", got)
	}
}

// TestScannerUnterminatedDiscardsBatch 同一输入块内出现未闭合区域时，本块已切出的语句一并丢弃：
// 宁可不执行也不把半段脚本交给服务端（调用方整体失败，导入链路会回滚事务）
func TestScannerUnterminatedDiscardsBatch(t *testing.T) {
	stmts, err := SplitStatements("SELECT 1; /* open", PgConfig, ';')
	if err == nil {
		t.Fatal("期望未闭合错误")
	}
	if len(stmts) != 0 {
		t.Fatalf("未闭合时应不产出语句, got=%q", stmts)
	}
}

// TestScannerPendingBlockGuard 未闭合复合块累积超 maxPendingBlockBytes 时必须降级重切，
// 不得无界缓存（超大 dump 单行值场景），也不得报错
func TestScannerPendingBlockGuard(t *testing.T) {
	long := strings.Repeat("a", 5<<20)
	stmts, err := SplitStatements("CREATE PROCEDURE p() BEGIN SELECT '"+long+"'; END;SELECT 2;", MysqlConfig, ';')
	if err != nil {
		t.Fatalf("累积超限应降级重切而非报错: %v", err)
	}
	if len(stmts) != 3 {
		t.Fatalf("语句数不符: got=%d", len(stmts))
	}
	if stmts[2] != "SELECT 2" {
		t.Fatalf("后续语句未正常切割: %q", stmts[2][:20])
	}
}
