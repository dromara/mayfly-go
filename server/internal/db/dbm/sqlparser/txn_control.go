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
	if !strings.HasPrefix(normalized, "SET") {
		return false
	}
	// 会话级事务开关的 SET 变体（SET autocommit / SET @@session.autocommit）。
	// 只在代码区找 AUTOCOMMIT：字符串字面量里的同名词（如 SET @x='AUTOCOMMIT'）不是事务开关，
	// 不剔除会让这种普通 SET 被判为事务控制语句，进而在迁移导入时被静默跳过
	return strings.Contains(maskStringLiterals(normalized), "AUTOCOMMIT")
}

// maskStringLiterals 把单/双引号字符串字面量（含定界符）替换为等长空白，供按代码区判关键字。
// 支持 ” / "" 双写转义与反斜杠转义（MySQL 默认 sql_mode）；未闭合时剩余部分整体视为字面量。
// 双引号在 ANSI_QUOTES 下是标识符而非字符串，一并掩掉对本判定无害——AUTOCOMMIT 只可能出现在代码区。
func maskStringLiterals(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\'' && s[i] != '"' {
			b.WriteByte(s[i])
			continue
		}
		quote := s[i]
		// 进入字面量：左侧定界符与内容一律置空白，直到右侧定界符
		b.WriteByte(' ')
		for i++; i < len(s); i++ {
			if s[i] == '\\' && i+1 < len(s) {
				// 反斜杠转义（\' \" \\）：两个字符都仍在字面量内
				b.WriteString("  ")
				i++
				continue
			}
			if s[i] != quote {
				b.WriteByte(' ')
				continue
			}
			if i+1 < len(s) && s[i+1] == quote { // '' / "" 双写转义仍在字面量内
				b.WriteString("  ")
				i++
				continue
			}
			b.WriteByte(' ')
			break
		}
	}
	return b.String()
}

// IsTxnControlStmt 按方言语义切割掩码后判断语句是否为事务控制语句（见 NormalizeStmtText、IsTxnControlStmtText）
func IsTxnControlStmt(splitter SQLSplitter, stmt string) bool {
	return IsTxnControlStmtText(NormalizeStmtText(splitter, stmt))
}

// implicitCommitStmtPrefixes 会隐式提交当前事务、但既不是 DDL 也不是事务控制语句的语句前缀（MySQL 语义）。
// 仅用于失败时「可能已提交、无法回滚」的告警判定，不参与执行路径。
var implicitCommitStmtPrefixes = []string{
	"LOCK ", "UNLOCK ", "ANALYZE ", "OPTIMIZE ", "REPAIR ",
}

// IsImplicitCommitStmtText 判断归一化后的语句是否为隐式提交语句
// （LOCK/UNLOCK TABLES、ANALYZE/OPTIMIZE/REPAIR TABLE）。
func IsImplicitCommitStmtText(normalized string) bool {
	for _, prefix := range implicitCommitStmtPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

// IsImplicitCommitStmt 按方言语义归一化后判断语句是否为隐式提交语句（与 IsTxnControlStmt 互补）。
func IsImplicitCommitStmt(splitter SQLSplitter, stmt string) bool {
	return IsImplicitCommitStmtText(NormalizeStmtText(splitter, stmt))
}
