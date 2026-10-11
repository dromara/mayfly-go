package starter

import (
	"context"
	"path/filepath"
	"testing"
)

// TestSqliteDsnAppliesPragmas busy 超时是这次连接策略的关键：锁冲突要排队而不是立刻失败，
// 而已带查询串的地址必须用 & 追加，否则造出的 DSN 打不开库
func TestSqliteDsnAppliesPragmas(t *testing.T) {
	for _, tc := range []struct {
		address  string
		contains string
	}{
		{"mayfly-go.db", "?_pragma=busy_timeout(5000)"},
		{"file:/data/a.db", "?_pragma=busy_timeout(5000)"},
		{"file:/data/a.db?_journal_mode=memory", "&_pragma=busy_timeout(5000)"},
	} {
		got := sqliteDsn(tc.address)
		if want := tc.address + tc.contains; got != want {
			t.Errorf("sqliteDsn(%q) = %q, want %q", tc.address, got, want)
		}
	}
}

// TestInitSqliteKeepsPoolAndBusyTimeout 连接池沿用配置值，且 busy 超时确实落到库上。
//
// 收成一条连接能让写入完全串行、避开所有锁冲突，但代价是事务内的读永远等不到第二条连接而挂死
// —— 基座那批不收 ctx 的读入口在业务事务链上大量存在，因此这里必须钉住「池不被改小」
func TestInitSqliteKeepsPoolAndBusyTimeout(t *testing.T) {
	db, err := initSQLite(DBConf{
		Address:      filepath.Join(t.TempDir(), "meta.db"),
		MaxIdleConns: 5,
		MaxOpenConns: 50,
	})
	if err != nil {
		t.Fatalf("initSQLite failed: %v", err)
	}

	var busy int
	if err := db.WithContext(context.Background()).Raw("PRAGMA busy_timeout").Scan(&busy).Error; err != nil {
		t.Fatalf("read busy_timeout failed: %v", err)
	}
	if busy != sqliteBusyTimeoutMs {
		t.Errorf("busy_timeout = %d, want %d", busy, sqliteBusyTimeoutMs)
	}

	sqlDb, err := db.DB()
	if err != nil {
		t.Fatalf("db handle failed: %v", err)
	}
	if limit := sqlDb.Stats().MaxOpenConnections; limit != 50 {
		t.Errorf("MaxOpenConnections = %d, want 50：连接池必须沿用配置，收成 1 会让事务内的读挂死", limit)
	}
}
