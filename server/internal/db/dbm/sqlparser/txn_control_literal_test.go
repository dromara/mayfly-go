package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTxnControlIgnoresAutoCommitInLiteral 事务控制判定只在代码区识别 AUTOCOMMIT：
// 字符串字面量里的同名词不是会话事务开关，若不剔除会被判为事务控制语句，
// 进而在迁移导入路径（transfer/db_transfer_import.go）被静默跳过。
func TestTxnControlIgnoresAutoCommitInLiteral(t *testing.T) {
	// 会话级事务开关：命中
	require.True(t, IsTxnControlStmtText("SET AUTOCOMMIT=0"))
	require.True(t, IsTxnControlStmtText("SET @@SESSION.AUTOCOMMIT=1"))

	// 普通 SET，只在字面量里出现同名文本：不得误判
	require.False(t, IsTxnControlStmtText("SET @X='AUTOCOMMIT'"))
	require.False(t, IsTxnControlStmtText("SET @MSG='PLEASE SET AUTOCOMMIT OFF'"))
	// '' 转义仍在字面量内，同样不得命中
	require.False(t, IsTxnControlStmtText("SET @X='A''AUTOCOMMIT'"))
	// 双引号字面量里的同名文本：不得误判
	require.False(t, IsTxnControlStmtText(`SET @MSG="AUTOCOMMIT"`))

	// 反斜杠转义（MySQL 默认 sql_mode）：字面量在 \' 处不结束，后面的 AUTOCOMMIT 仍属代码区 → 命中
	require.True(t, IsTxnControlStmtText(`SET @MSG='IT\'S', AUTOCOMMIT=0`))
	// 双引号字符串内含单引号：只掩双引号字面量，后面的 AUTOCOMMIT 必须命中
	require.True(t, IsTxnControlStmtText(`SET @MSG="IT'S", AUTOCOMMIT=0`))

	// 普通 SET 无 AUTOCOMMIT
	require.False(t, IsTxnControlStmtText("SET NAMES UTF8MB4"))

	// 精确事务控制词（整体相等，非前缀）
	require.True(t, IsTxnControlStmtText("BEGIN"))
	require.True(t, IsTxnControlStmtText("COMMIT"))
	require.True(t, IsTxnControlStmtText("ROLLBACK"))
}

// TestIsImplicitCommitStmtText 隐式提交语句（非 DDL/事务控制）识别的正反例：
// 这些语句会让「失败整体回滚」静默失效，失败告警必须能识别到。
func TestIsImplicitCommitStmtText(t *testing.T) {
	for _, s := range []string{
		"LOCK TABLES T WRITE",
		"UNLOCK TABLES",
		"ANALYZE TABLE T",
		"OPTIMIZE TABLE T",
		"REPAIR TABLE T",
	} {
		require.True(t, IsImplicitCommitStmtText(s), s)
	}
	for _, s := range []string{
		"SELECT 1",
		"INSERT INTO T VALUES (1)",
		"BEGIN",
		"COMMIT",
		"LOCKED_ACCOUNTS", // 前缀必须带空格，避免误伤同前缀词
	} {
		require.False(t, IsImplicitCommitStmtText(s), s)
	}
}
