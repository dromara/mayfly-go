package sqlparser

import (
	"strings"
)

// NormalizeStmtText 归一化语句文本，供「按语句整体文本判定语句类型」使用：
// 掩码普通注释（字面量保留原文、可执行注释解壳）→ 分号转空白 → 折叠空白 → 转大写。
//
// 语句切割保留注释原文后，一条语句可能以注释开头（dump 分段注释头会与其后的 BEGIN 同属一条：
// "-- Data: t\n-- ----\nBEGIN"），直接 TrimSpace 或按字符前缀匹配都会误判，故统一走此归一化。
// 结果形态示例："-- c\nbegin ;" -> "BEGIN"，"INSERT INTO t VALUES ('BEGIN')" -> "INSERT INTO T VALUES ('BEGIN')"
func NormalizeStmtText(splitter SQLSplitter, stmt string) string {
	masked := splitter.MaskComments(stmt)
	// 分号替换为空白而非删除：兼容 "BEGIN ;"、"COMMIT;WORK" 等书写形式，同时不粘连相邻词
	return strings.ToUpper(strings.Join(strings.Fields(strings.ReplaceAll(masked, ";", " ")), " "))
}

// txnControlStmtTexts 事务控制语句的归一化文本形态。
// 用「整体相等」而非前缀匹配：以 BEGIN 开头的 PL/SQL 块（BEGIN ... END;）归一化后不等于 "BEGIN"，不会被误判
var txnControlStmtTexts = map[string]struct{}{
	"BEGIN": {}, "BEGIN WORK": {}, "BEGIN TRANSACTION": {},
	"START TRANSACTION": {}, "START TRANSACTION READ ONLY": {}, "START TRANSACTION READ WRITE": {},
	"COMMIT": {}, "COMMIT WORK": {},
	"ROLLBACK": {}, "ROLLBACK WORK": {},
}

// IsTxnControlStmtText 判断归一化后的语句文本是否为事务控制语句。
//
// 为什么重要：显式管理事务的链路（数据库导入、SQL文件执行）中，这类语句一旦被执行，
// 部分数据库会隐式提交当前事务（如 mysql 执行 BEGIN/COMMIT/LOCK TABLES 均会提交），
// 使「失败回滚」的承诺静默失效，失败后残留部分数据且无法通过回滚清理。
// 注意："SET IDENTITY_INSERT ..."（mssql/dm 自增列导入前置语句）不含 AUTOCOMMIT，不受影响
func IsTxnControlStmtText(normalized string) bool {
	if _, ok := txnControlStmtTexts[normalized]; ok {
		return true
	}
	// SET autocommit / SET @@session.autocommit 等变体：会话级事务开关，同样改变事务边界
	return strings.HasPrefix(normalized, "SET") && strings.Contains(normalized, "AUTOCOMMIT")
}

// IsTxnControlStmt 按方言语义切割掩码后判断语句是否为事务控制语句（见 NormalizeStmtText、IsTxnControlStmtText）
func IsTxnControlStmt(splitter SQLSplitter, stmt string) bool {
	return IsTxnControlStmtText(NormalizeStmtText(splitter, stmt))
}
