package api

import (
	"testing"
	"time"
)

// TestScanShouldStop 凑批循环退出判据的三条件边界：
// 任一条件命中即停，全部不命中则继续扫——预算条件保证稀疏匹配不会单请求扫完整库
func TestScanShouldStop(t *testing.T) {
	budget := 250 * time.Millisecond

	cases := []struct {
		name    string
		matched int64
		target  int64
		cursor  uint64
		elapsed time.Duration
		want    bool
	}{
		{name: "凑满目标数即停", matched: 250, target: 250, cursor: 12345, elapsed: time.Millisecond, want: true},
		{name: "游标归零即停", matched: 3, target: 250, cursor: 0, elapsed: time.Millisecond, want: true},
		{name: "预算耗尽即停", matched: 3, target: 250, cursor: 12345, elapsed: budget, want: true},
		{name: "预算内且未凑满且游标未归零则继续", matched: 3, target: 250, cursor: 12345, elapsed: budget - time.Millisecond, want: false},
		{name: "零匹配但游标未归零且预算内仍继续", matched: 0, target: 250, cursor: 999, elapsed: 0, want: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := scanShouldStop(c.matched, c.target, c.cursor, c.elapsed, budget); got != c.want {
				t.Fatalf("scanShouldStop(%d, %d, %d, %v) = %v, want %v", c.matched, c.target, c.cursor, c.elapsed, got, c.want)
			}
		})
	}
}
