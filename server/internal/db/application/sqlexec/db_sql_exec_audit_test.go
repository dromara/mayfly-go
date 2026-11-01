package sqlexec

// SQL 审计文本的截断语义：长语句必须留下「不完整但知情」的记录，而不是写不进去后静默消失。

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"mayfly-go/internal/db/application/dto"
)

func TestTruncateForAudit(t *testing.T) {
	// 未超限：原样返回，不加任何标记
	short := "update t set a = 1 where id = 2"
	if got := truncateForAudit(short, 128); got != short {
		t.Fatalf("short text must stay untouched, got %q", got)
	}

	// 恰好等于上限：同样不截断（边界不应被误伤）
	exact := strings.Repeat("a", 64)
	if got := truncateForAudit(exact, 64); got != exact {
		t.Fatalf("text of exactly maxBytes must stay untouched, got %q", got)
	}

	// 超限：保留前 maxBytes 字节，并追加带原始长度的标记
	long := strings.Repeat("a", 100)
	got := truncateForAudit(long, 32)
	if !strings.HasPrefix(got, strings.Repeat("a", 32)) {
		t.Fatalf("truncated text must keep the first maxBytes bytes, got %q", got)
	}
	if !strings.Contains(got, "kept 32 of 100 bytes") {
		t.Fatalf("marker must carry the original size, got %q", got)
	}
	if len(got) <= 32 {
		t.Fatalf("marker must be appended after the kept bytes, got len=%d", len(got))
	}

	// 多字节字符：截断点落在字符中间时必须回退到字符边界，结果仍是合法 UTF-8。
	// 半个汉字落库会报错或显示乱码，等于把「审计不完整」升级成「审计报错」
	chinese := strings.Repeat("中", 20) // 每个 3 字节
	cutCN := truncateForAudit(chinese, 32)
	if !utf8.ValidString(cutCN) {
		t.Fatalf("truncated multi-byte text must stay valid UTF-8, got %q", cutCN)
	}
	if !strings.HasPrefix(cutCN, strings.Repeat("中", 10)) {
		t.Fatalf("truncation must stop on a rune boundary (30 bytes = 10 chars), got prefix %q", cutCN)
	}
}

// TestCreateSQLExecRecordTruncatesSQL 记录构造点是唯一写入口：
// 无论执行、回滚生成还是工单回放，读到的都应是同一份「已按上限截断并标注」的文本。
func TestCreateSQLExecRecordTruncatesSQL(t *testing.T) {
	huge := strings.Repeat("x", maxAuditSQLBytes+10)
	record := createSQLExecRecord(context.Background(), &dto.DbSQLExecReq{DbId: 1, Db: "shop"}, huge)
	if !strings.Contains(record.SQL, "[audit] truncated") {
		t.Fatal("oversized SQL must be truncated with an explicit marker before hitting the column")
	}
	if !strings.Contains(record.SQL, "of "+strconv.Itoa(len(huge))+" bytes") {
		t.Fatalf("marker must report the original size, got %q", record.SQL[len(record.SQL)-80:])
	}

	// 正常长度不产生任何标记
	normal := createSQLExecRecord(context.Background(), &dto.DbSQLExecReq{DbId: 1, Db: "shop"}, "select 1")
	if normal.SQL != "select 1" {
		t.Fatalf("normal SQL must be stored verbatim, got %q", normal.SQL)
	}
}
