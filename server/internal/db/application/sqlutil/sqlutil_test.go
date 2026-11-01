package sqlutil

// 中立共享 helper 的单测。
//
// BriefReason 被 explain（父包）与 review/rollback（sqlexec）共用：说明列里塞进数据库
// 原始错误全文是读不完的墙，多行只留首行、超长截断，完整原因留给服务端日志。

import (
	"errors"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
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

// TestExportSelects 导出判定语句：逐表一条、按方言引用符包裹、空名跳过。
//
// 逐表一条而不是拼多表 JOIN：策略里的「目标表」事实必须与逐表导出的实际行为一致，
// 拼成一条会让命中集合随拼接方式漂移。
func TestExportSelects(t *testing.T) {
	mysqlQuoter := dbi.Quoter{Prefix: []byte("`")[0], Suffix: []byte("`")[0], IsReserved: dbi.AlwaysReserve}
	got := ExportSelects(mysqlQuoter, []string{"t_order", " order_info ", "", "  "})
	if len(got) != 2 {
		t.Fatalf("blank names must be skipped, got %v", got)
	}
	if got[0] != "SELECT * FROM `t_order`" || got[1] != "SELECT * FROM `order_info`" {
		t.Fatalf("table names must be trimmed and quoted, got %v", got)
	}

	// 默认引用符（双引号）方言
	pg := ExportSelects(dbi.DefaultQuoter, []string{"public.users"})
	if len(pg) != 1 || pg[0] != `SELECT * FROM "public"."users"` {
		t.Fatalf("qualified names must be quoted per part, got %v", pg)
	}

	if empty := ExportSelects(dbi.DefaultQuoter, nil); len(empty) != 0 {
		t.Fatalf("no tables must yield no statements, got %v", empty)
	}
}
