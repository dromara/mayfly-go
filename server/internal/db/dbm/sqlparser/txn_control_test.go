package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// TestNormalizeStmtText 归一化文本：掩码注释、分号转空白、折叠空白、转大写，字面量保留原文
func TestNormalizeStmtText(t *testing.T) {
	splitter := NewSplitter(tokenizer.MysqlConfig)
	cases := map[string]string{
		"BEGIN":                        "BEGIN",
		"  begin ; ":                   "BEGIN",
		"-- c\nBEGIN":                  "BEGIN",
		"/* head */ commit;":           "COMMIT",
		"BEGIN\nWORK":                  "BEGIN WORK",
		"SET @@session.autocommit = 1": "SET @@SESSION.AUTOCOMMIT = 1",
		// 字面量内容不得被掩码/删除（仅大小写被统一，判定只用整体相等/前缀，不影响正确性）
		"INSERT INTO t VALUES ('-- c\\nBEGIN')": "INSERT INTO T VALUES ('-- C\\NBEGIN')",
		"-- 仅注释":                                "",
		"":                                      "",
		"/*!40101 SET autocommit=0 */":          "SET AUTOCOMMIT=0",
		"/*!40000 ALTER TABLE `t` DISABLE KEYS */": "ALTER TABLE `T` DISABLE KEYS",
	}
	for sql, want := range cases {
		assert.Equalf(t, want, NormalizeStmtText(splitter, sql), "归一化结果不符: %q", sql)
	}
}

// TestIsTxnControlStmtText 事务控制语句按「整体相等」判定：块语句与相似前缀词不得误判
func TestIsTxnControlStmtText(t *testing.T) {
	splitter := NewSplitter(tokenizer.MysqlConfig)
	isTxn := func(stmt string) bool { return IsTxnControlStmt(splitter, stmt) }

	for _, stmt := range []string{
		"BEGIN", "begin;", "COMMIT", "commit work;", "ROLLBACK", "START TRANSACTION",
		"start transaction read only;", "SET autocommit=0", "SET @@GLOBAL.autocommit = 1",
		"-- 分段注释\nBEGIN", "/* head */ COMMIT;", "/*!40101 SET autocommit=0 */",
	} {
		assert.Truef(t, isTxn(stmt), "应识别为事务控制语句: %q", stmt)
	}

	for _, stmt := range []string{
		"BEGINNING", "BEGINX", "SELECT 1",
		// 以 BEGIN 开头的匿名块：整体文本不等于 BEGIN，属业务语句（误判即静默丢语句）
		"BEGIN INSERT INTO t VALUES (1); END;",
		"SET IDENTITY_INSERT [t] ON",
		"INSERT INTO t VALUES ('COMMIT;')",
		"-- c\nDELETE FROM t",
	} {
		assert.Falsef(t, isTxn(stmt), "不应识别为事务控制语句: %q", stmt)
	}
}
