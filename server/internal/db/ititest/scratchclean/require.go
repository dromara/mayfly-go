package scratchclean

import (
	"os"
	"strings"
	"testing"
)

// requiredDialectsEnv 声明本次运行「必须真实执行」的方言（逗号分隔，如 mssql,oracle）。
// 列出的方言若连接不可用则判定为失败，而非静默跳过；未列出的方言保持不可用时跳过。
// 对齐业界做法：CI 用 docker compose 起齐所有方言后设此变量，杜绝「永远跳过 = 误以为通过」。
const requiredDialectsEnv = "MAYFLY_IT_REQUIRED_DIALECTS"

// DialectRequired 报告指定方言是否被要求必须真实执行。
func DialectRequired(dialect string) bool {
	for _, d := range strings.Split(os.Getenv(requiredDialectsEnv), ",") {
		if strings.EqualFold(strings.TrimSpace(d), dialect) {
			return true
		}
	}
	return false
}

// SkipOrRequire 处理方言实例不可用的情形：被要求则 t.Fatalf，否则 t.Skipf。
// reason 应包含底层错误，便于区分「未启动」与「连不上」。
func SkipOrRequire(t testing.TB, dialect, reason string) {
	t.Helper()
	if DialectRequired(dialect) {
		t.Fatalf("required dialect [%s] unavailable: %s", dialect, reason)
	}
	t.Skipf("dialect [%s] unavailable, skipping: %s", dialect, reason)
}
