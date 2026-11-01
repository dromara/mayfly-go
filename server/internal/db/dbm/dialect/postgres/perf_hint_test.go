package postgres

// PG 慢查询不可用的三种真实成因要给出不同步骤，用户才知道改配置还是装扩展。

import (
	"strings"
	"testing"
)

func TestPgSlowQueryHint(t *testing.T) {
	notInstalled := pgSlowQueryHint(`pq: relation "pg_stat_statements" does not exist`)
	if !strings.Contains(notInstalled, "CREATE EXTENSION") || !strings.Contains(notInstalled, "shared_preload_libraries") {
		t.Fatalf("not installed must mention both preload and CREATE EXTENSION, got %q", notInstalled)
	}

	notPreloaded := pgSlowQueryHint("pg_stat_statements must be loaded via shared_preload_libraries")
	if !strings.Contains(notPreloaded, "shared_preload_libraries") || strings.Contains(notPreloaded, "CREATE EXTENSION") {
		t.Fatalf("not preloaded must only ask for config, got %q", notPreloaded)
	}

	denied := pgSlowQueryHint("pq: permission denied for view pg_stat_statements")
	if !strings.Contains(denied, "pg_read_all_stats") {
		t.Fatalf("permission case must suggest pg_read_all_stats, got %q", denied)
	}

	// 未知原因不给步骤：宁可不提示，也不给一条误导人的操作
	if hint := pgSlowQueryHint("pq: connection reset by peer"); hint != "" {
		t.Fatalf("unknown reason must not fabricate steps, got %q", hint)
	}
}
