package sqlutil

// 中立共享 helper 的单测。
//
// BriefReason 被 explain（父包）与 review/rollback（sqlexec）共用：说明列里塞进数据库
// 原始错误全文是读不完的墙，多行只留首行、超长截断，完整原因留给服务端日志。

import (
	"errors"
	"strings"
	"testing"
)

func TestBriefReasonKeepsItReadable(t *testing.T) {
	long := errors.New("Error 1064 (42000): You have an error in your SQL syntax; check the manual\nnear 'DELETE FROM t' at line 2\n" + strings.Repeat("x", 400))
	got := BriefReason(long)
	if strings.Contains(got, "\n") {
		t.Fatalf("the reason must be a single line, got %q", got)
	}
	if !strings.HasPrefix(got, "Error 1064") {
		t.Fatalf("the head of the message carries the verdict, got %q", got)
	}
	if runes := []rune(got); len(runes) > 161 {
		t.Fatalf("the reason must be capped, got %d runes", len(runes))
	}

	if BriefReason(nil) != "" {
		t.Fatal("a nil error must render as an empty reason rather than panic")
	}
}
