package transfer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "mayfly-go/internal/db/dbm" // 触发各方言注册，使用生产同款方言切割器
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// 导入切割全链路单测（不依赖真实数据库）
//
// 背景：导入链路为「方言切割器 SplitSQL → shouldSkipImportStmt 过滤 → 逐条 TxExec → 批级提交」，
// 其中过滤环节的输入是切割器产出的**语句原文**（保留注释），切割规则的任何调整都会直接影响
// 「哪些语句被执行、以什么内容被执行」。这条链路出错的表现是静默丢语句/静默多提交/半提交残留数据，
// 属于数据安全事故级别，故在此把 dump 产物的真实形态固化为断言。
//
// 断言口径：executed 必须与「应被数据库执行的语句」逐条、按序、原文一致。

// dumpExecuted 驱动生产同一条「切割 → 过滤」决策链（iterImportStmts），返回将被执行的语句列表
// （真实链路中每条 executed 都会 TxExec，此处不连库，只验证决策正确性）
func dumpExecuted(t *testing.T, dbType dbi.DbType, sql string) []string {
	t.Helper()
	dialect := dbi.GetDialect(dbType)
	require.NotNilf(t, dialect, "方言 [%s] 未注册", dbType)
	splitter := dialect.GetSQLSplitter()
	require.NotNil(t, splitter)

	var executed []string
	err := iterImportStmts(splitter, strings.NewReader(sql), func(stmt string) error {
		executed = append(executed, stmt)
		return nil
	})
	require.NoErrorf(t, err, "切割 dump 失败: %+v", err)
	return executed
}

// mayflyDumpMysql 平台自有 dump 产物真实形态：注释分段头紧贴其后的语句，数据段以 BEGIN;/COMMIT; 包装
// （见 db_dump.go 与 dbi.DefaultDumpHelper.BeforeInsert/AfterInsert）
const mayflyDumpMysql = `
-- ----------------------------
-- Dump Platform: mayfly-go
-- ----------------------------

-- ----------------------------
-- Table structure: t_user
-- ----------------------------
DROP TABLE IF EXISTS t_user;
CREATE TABLE t_user (id bigint NOT NULL AUTO_INCREMENT, name varchar(64), PRIMARY KEY (id));

-- ----------------------------
-- Data: t_user
-- ----------------------------
BEGIN;
INSERT INTO t_user (id,name) VALUES (1,'a'),(2,'b');
INSERT INTO t_user (id,name) VALUES (3,'c');
COMMIT;

-- ----------------------------
-- Table Index: t_user
-- ----------------------------
CREATE INDEX idx_name ON t_user (name);
`

// TestImportDumpFiltersTxnWrapperOfMayflyDump 平台 dump 的事务包装必须被过滤，业务语句一条不丢
//
// 回归的缺陷：切割保留注释原文后，"-- Data: t_user\n...\nBEGIN" 成为一条语句，按文本前缀判定得不出
// BEGIN，该语句会被真正执行——mysql 执行 BEGIN 会隐式提交导入侧已开启的批次事务，
// 使「批级提交 + 失败回滚」失效，失败后残留部分数据且无法通过回滚清理。
func TestImportDumpFiltersTxnWrapperOfMayflyDump(t *testing.T) {
	got := dumpExecuted(t, "mysql", mayflyDumpMysql)

	require.Len(t, got, 5, "应执行且仅执行5条业务语句（DROP/CREATE/2×INSERT/CREATE INDEX）")
	// 首条保留了分段注释头 + DROP（注释不得被丢弃，也不能吞掉后面的语句）
	assert.Contains(t, got[0], "DROP TABLE IF EXISTS t_user")
	assert.Contains(t, got[0], "-- Table structure: t_user")
	// DDL/索引语句完整
	assert.Equal(t, "CREATE TABLE t_user (id bigint NOT NULL AUTO_INCREMENT, name varchar(64), PRIMARY KEY (id))", got[1])
	assert.Equal(t, "INSERT INTO t_user (id,name) VALUES (1,'a'),(2,'b')", got[2])
	assert.Equal(t, "INSERT INTO t_user (id,name) VALUES (3,'c')", got[3])
	assert.Contains(t, got[4], "CREATE INDEX idx_name ON t_user (name)")
	// 原文完整性：所有应执行的语句都按原样出现，未被改写或截断
	assert.True(t, strings.Contains(got[4], "-- Table Index: t_user"), "注释与语句合并后原文被改写")
}

// TestImportDumpFiltersBareTxnStmts 无注释前缀的事务语句同样过滤（切割器不同版本行为差异下的兜底）
func TestImportDumpFiltersBareTxnStmts(t *testing.T) {
	got := dumpExecuted(t, "mysql", "BEGIN;\nINSERT INTO t VALUES (1);\nCOMMIT;\n")
	require.Equal(t, []string{"INSERT INTO t VALUES (1)"}, got)
}

// TestImportDumpMysqldumpThirdParty 第三方 mysqldump 产物：可执行注释按内容判定，
// 仅事务相关的被过滤，其余（含 SET NAMES / DISABLE KEYS / LOCK TABLES）必须原样交给数据库
func TestImportDumpMysqldumpThirdParty(t *testing.T) {
	dump := "/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;\n" +
		"/*!40101 SET NAMES utf8mb4 */;\n" +
		"/*!40101 SET autocommit=0 */;\n" + // 会话事务开关：由导入器管理事务，须过滤
		"DROP TABLE IF EXISTS `t`;\n" +
		"CREATE TABLE `t` (id int) /*!50100 ENGINE=InnoDB */;\n" +
		"LOCK TABLES `t` WRITE;\n" +
		"/*!40000 ALTER TABLE `t` DISABLE KEYS */;\n" +
		"INSERT INTO `t` VALUES (1),(2);\n" +
		"UNLOCK TABLES;\n"

	got := dumpExecuted(t, "mysql", dump)

	joined := strings.Join(got, "\n")
	assert.NotContains(t, joined, "autocommit", "SET autocommit 未被过滤，将破坏导入事务语义")
	// 其余可执行注释属业务/会话设置语句，不得被静默丢弃
	assert.Contains(t, joined, "SET NAMES utf8mb4", "可执行注释内容被误过滤")
	assert.Contains(t, joined, "DISABLE KEYS", "可执行注释内容被误过滤")
	assert.Contains(t, joined, "INSERT INTO `t` VALUES (1),(2)", "数据语句丢失")
	// CREATE TABLE 尾部的版本门控注释必须与建表语句同属一条（否则表引擎/字符集丢失）
	createIdx := -1
	for i, stmt := range got {
		if strings.Contains(stmt, "CREATE TABLE") {
			createIdx = i
		}
	}
	require.NotEqual(t, -1, createIdx, "CREATE TABLE 语句丢失")
	assert.Contains(t, got[createIdx], "ENGINE=InnoDB", "建表语句尾部的可执行注释与建表语句被错误切割")
}

// TestImportDumpLiteralContainingTxnKeyword 数据字面量中的事务词/注释符不得触发过滤
//
// 误过滤的后果是静默丢数据行，比不过滤更危险，故单独固化。
func TestImportDumpLiteralContainingTxnKeyword(t *testing.T) {
	dump := "INSERT INTO t (c) VALUES ('-- not comment\\nBEGIN');\n" +
		"INSERT INTO t (c) VALUES ('COMMIT;');\n" +
		"INSERT INTO t (c) VALUES ('/* BEGIN */');\n"

	got := dumpExecuted(t, "mysql", dump)
	require.Len(t, got, 3, "字面量含事务词/注释符的数据行被误过滤")
	for _, stmt := range got {
		assert.True(t, strings.HasPrefix(stmt, "INSERT INTO t"), "语句被改写: %q", stmt)
	}
}

// TestImportDumpCommentOnlyTextNotExecuted 纯注释文本不得作为语句交给数据库执行
// （空/纯注释语句在多数驱动上会报语法错误或「空语句」错误，导致导入中断）
func TestImportDumpCommentOnlyTextNotExecuted(t *testing.T) {
	for _, sql := range []string{
		"-- 仅注释\n",
		"/* 仅块注释 */",
		"# 仅注释\n-- 另一段\n",
	} {
		got := dumpExecuted(t, "mysql", sql)
		assert.Emptyf(t, got, "纯注释文本被当作语句执行: %q -> %q", sql, got)
	}
}

// TestImportDumpUnterminatedFailsLoud 文件被截断（未闭合引号/注释）必须切割报错而非静默导入半份数据
//
// 场景：zip 读取有 10MB 上限、上传中断等导致内容不完整时，末尾未闭合语句会把后续内容并成一条；
// 若不报错，将执行一条被污染的语句并可能提交已成功部分。
func TestImportDumpUnterminatedFailsLoud(t *testing.T) {
	cases := map[string]string{
		"未闭合字符串": "INSERT INTO t VALUES (1);\nINSERT INTO t VALUES ('abc",
		"未闭合块注释": "INSERT INTO t VALUES (1);\n/* pending\nINSERT INTO t VALUES (2);",
	}
	for name, sql := range cases {
		dialect := dbi.GetDialect("mysql")
		err := dialect.GetSQLSplitter().SplitSQL(strings.NewReader(sql), func(string) error { return nil })
		var ue *tokenizer.UnterminatedError
		require.ErrorAsf(t, err, &ue, "[%s] 未闭合内容必须返回 UnterminatedError, got=%v", name, err)
		assert.Equalf(t, 2, ue.Line, "[%s] 切割错误行号定位不准", name)
		// 业务错误转换：转为国际化文案（排障依赖行号与区域类型），不得返回 nil 或吞掉错误
		bizErr := sqlparser.SplitError(t.Context(), err)
		require.Error(t, bizErr)
		assert.Contains(t, bizErr.Error(), "2", "转换后的错误缺少行号信息")
	}
}

// TestImportDumpNoDataLossOnLargeScript 大文件（多行值批量插入 + 每表事务包装）语句数与内容零丢失
func TestImportDumpNoDataLossOnLargeScript(t *testing.T) {
	var sb strings.Builder
	const tables, rowsPerTable = 20, 60 // 超过 importStmtBatchSize，覆盖多次批级提交
	expected := 0
	for tbl := 1; tbl <= tables; tbl++ {
		fmt.Fprintf(&sb, "-- ----------------------------\n-- Data: t_%d\n-- ----------------------------\nBEGIN;\n", tbl)
		for row := 1; row <= rowsPerTable; row++ {
			fmt.Fprintf(&sb, "INSERT INTO t_%d (id,name) VALUES (%d,'v-%d');\n", tbl, row, tbl*1000+row)
			expected++
		}
		sb.WriteString("COMMIT;\n\n")
	}

	got := dumpExecuted(t, "mysql", sb.String())
	require.Len(t, got, expected, "大 dump 语句数与预期不符（丢失或多执行）")
	for i, stmt := range got {
		assert.Truef(t, strings.HasPrefix(stmt, "INSERT INTO t_"), "第%d条应为INSERT, got=%q", i+1, stmt)
	}
}

// TestImportDumpPlSqlBlockNotTreatedAsTxnStart 以 BEGIN 开头的存储过程/匿名块不能被当成事务开始语句过滤
//
// 过滤判定用的是「归一化后整体等于 BEGIN」，块体内容使其不等于 BEGIN，故不会被静默丢弃。
func TestImportDumpPlSqlBlockNotTreatedAsTxnStart(t *testing.T) {
	block := "-- 自定义段\nBEGIN\n  INSERT INTO t VALUES (1);\n  INSERT INTO t VALUES (2);\nEND;"
	got := dumpExecuted(t, "oracle", block)
	require.Len(t, got, 1, "PL/SQL 块被错误切割")
	assert.Contains(t, got[0], "END", "PL/SQL 块体不完整（END 丢失将导致执行失败）")
	assert.Contains(t, got[0], "INSERT INTO t VALUES (2)")
}

// TestImportDumpPgFunctionDollarQuote pg 函数体（dollar-quote）内的 BEGIN/END/分号不得影响切割与过滤
func TestImportDumpPgFunctionDollarQuote(t *testing.T) {
	dump := "CREATE OR REPLACE FUNCTION sync_t() RETURNS void AS $$\n" +
		"BEGIN\n" +
		"  DELETE FROM t WHERE id < 0;\n" +
		"  INSERT INTO t SELECT 1, 'x';\n" +
		"END;\n" +
		"$$ LANGUAGE plpgsql;\n" +
		"INSERT INTO t VALUES (2, 'y');\n"

	got := dumpExecuted(t, "postgres", dump)
	require.Len(t, got, 2, "函数体必须整体为一条语句")
	assert.Contains(t, got[0], "$$ LANGUAGE plpgsql", "函数体切割不完整")
	assert.Equal(t, "INSERT INTO t VALUES (2, 'y')", got[1])
}
