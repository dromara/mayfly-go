package cache

import (
	"testing"
	"time"
)

// itemOf 读取缓存项原始信息，用于断言过期时间是否被刷新
func itemOf(t *testing.T, lc *LocalCache, key string) *Item {
	t.Helper()
	items := lc.tm.Items()
	item, ok := items[key]
	if !ok {
		t.Fatalf("key %q not found in cache", key)
	}
	return item
}

func TestLocalCacheIncrWithTTL(t *testing.T) {
	lc := NewLocalCache()

	for i := int64(1); i <= 3; i++ {
		val, err := lc.IncrWithTTL("it:counter", time.Hour)
		if err != nil {
			t.Fatalf("IncrWithTTL error: %v", err)
		}
		if val != i {
			t.Fatalf("expected %d, got %d", i, val)
		}
	}

	item := itemOf(t, lc, "it:counter")
	if item.Expiration != int64(time.Hour) {
		t.Fatalf("expected expiration %d, got %d", int64(time.Hour), item.Expiration)
	}
	if item.Expired() {
		t.Fatal("item should not be expired")
	}
}

// TestLocalCacheIncrWithTTLRefreshesAccessTime 验证自增会续期。
// TimedCache 的过期判定基于 AccessTime，而 Increment 不更新 AccessTime，
// 若沿用 Incr，长期自增的计数器仍会在首次写入的过期时间到达时被清除
func TestLocalCacheIncrWithTTLRefreshesAccessTime(t *testing.T) {
	lc := NewLocalCache()

	if _, err := lc.IncrWithTTL("it:ttl", time.Hour); err != nil {
		t.Fatalf("IncrWithTTL error: %v", err)
	}
	first := itemOf(t, lc, "it:ttl").AccessTime

	time.Sleep(10 * time.Millisecond)

	val, err := lc.IncrWithTTL("it:ttl", time.Hour)
	if err != nil {
		t.Fatalf("IncrWithTTL error: %v", err)
	}
	if val != 2 {
		t.Fatalf("expected value to be preserved across ttl refresh, got %d", val)
	}

	access := itemOf(t, lc, "it:ttl").AccessTime
	if access <= first {
		t.Fatalf("expected access time refreshed, first=%d access=%d", first, access)
	}
}

// TestLocalCacheIncrKeepsOriginalExpiration 记录 Incr 的既有语义：只自增不续期。
// 这正是告警计数器必须改用 IncrWithTTL 的原因
func TestLocalCacheIncrKeepsOriginalExpiration(t *testing.T) {
	lc := NewLocalCache()

	if _, err := lc.Incr("it:plain"); err != nil {
		t.Fatalf("Incr error: %v", err)
	}
	first := itemOf(t, lc, "it:plain").AccessTime

	time.Sleep(10 * time.Millisecond)

	if _, err := lc.Incr("it:plain"); err != nil {
		t.Fatalf("Incr error: %v", err)
	}
	if access := itemOf(t, lc, "it:plain").AccessTime; access != first {
		t.Fatalf("Incr should not refresh access time, first=%d access=%d", first, access)
	}
}
