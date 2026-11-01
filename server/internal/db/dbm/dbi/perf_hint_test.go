package dbi

// 「慢查询不可用」除了原因还要带启用步骤：只报驱动原文会让用户不知道下一步做什么。

import (
	"errors"
	"strings"
	"testing"
)

func TestPerfUnavailableWithHint(t *testing.T) {
	err := NewPerfUnavailableWithHint("slow_query", `pq: relation "pg_stat_statements" does not exist`, "启用步骤：CREATE EXTENSION")

	// 仍要满足 errors.Is(ErrPerfUnavailable)：应用层靠它区分「不可用」与「真故障」
	if !errors.Is(err, ErrPerfUnavailable) {
		t.Fatal("hint error must still match ErrPerfUnavailable")
	}
	if hint := PerfHint(err); hint != "启用步骤：CREATE EXTENSION" {
		t.Fatalf("hint = %q", hint)
	}
	// 原因原文必须保留在 Error() 里（应用层取尾部括号内容展示）
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("reason lost: %s", err.Error())
	}

	// 不带 hint 的不可用错误：PerfHint 为空，不能凭空造步骤
	if hint := PerfHint(NewPerfUnavailable("lock_wait", "not supported")); hint != "" {
		t.Fatalf("plain unavailable must have empty hint, got %q", hint)
	}
	if hint := PerfHint(errors.New("boom")); hint != "" {
		t.Fatalf("unrelated error must have empty hint, got %q", hint)
	}
}
