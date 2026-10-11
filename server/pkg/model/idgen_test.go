package model

import (
	"sync"
	"testing"
)

// TestTimestampIdStrictlyIncreasing 同秒连续发号必须严格递增：
// 批量新增在同一秒内落库时，重复值会以 duplicate key 打断整批写入
func TestTimestampIdStrictlyIncreasing(t *testing.T) {
	ids := make([]uint64, 0, 1000)
	for i := 0; i < 1000; i++ {
		ids = append(ids, TimestampId())
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Fatalf("timestamp id not strictly increasing at %d: %d <= %d", i, ids[i], ids[i-1])
		}
	}
}

// TestTimestampIdConcurrentUnique 并发发号不得重号：发号器带着进程内状态，
// 无同步的实现会在批量导入时成对撞主键
func TestTimestampIdConcurrentUnique(t *testing.T) {
	const workers, per = 16, 500
	var wg sync.WaitGroup
	ch := make(chan uint64, workers*per)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < per; i++ {
				ch <- TimestampId()
			}
		}()
	}
	go func() { wg.Wait(); close(ch) }()

	seen := make(map[uint64]bool, workers*per)
	for id := range ch {
		if seen[id] {
			t.Fatalf("duplicate id generated concurrently: %d", id)
		}
		seen[id] = true
	}
	if len(seen) != workers*per {
		t.Fatalf("got %d ids, want %d", len(seen), workers*per)
	}
}
