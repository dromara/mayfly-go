package sync

import (
	"time"

	"mayfly-go/internal/db/domain/entity"
)

// resolveCursorInclusivity 决定实际生效的水位边界比较符（`>` 或 `>=`）。
//
// Auto 分支按同步模式给出「业界默认安全配置」，避免用户不懂交付语义时误配：
//   - 增量追加(1) / 全量刷新(3)：目标不做 UPSERT，重发即重复行 → 强制 Exclusive
//   - 增量合并(2) / 全量对账(4/5)：目标 UPSERT/DELETE NOT IN 天然幂等 → Inclusive（防同秒漏拉）
//   - 数据校验(6)：不写目标 → Exclusive（inclusive 无意义）
//
// 用户显式覆写时按覆写值：Save 校验会拒绝「Append + Inclusive」「无 PK 目标 + Inclusive」等
// 语义矛盾组合，见 validateCursorSemantics。
func resolveCursorInclusivity(mode entity.DataSyncMode, configured entity.CursorInclusivity) bool {
	switch configured {
	case entity.CursorInclusivityExclusive:
		return false
	case entity.CursorInclusivityInclusive:
		return true
	}
	// Auto
	return mode == entity.DataSyncModeIncrementalMerge ||
		mode == entity.DataSyncModeIncrementalSoftDel ||
		mode == entity.DataSyncModeIncrementalHardDel
}

// interruptibleSleep 睡眠 d，期间每 tickInterval 检查一次 keepGoing。
// keepGoing 返回 false 时立刻返回 false，不再等剩余时长——用于响应停止请求 / 锁被抢占。
//
// d<=0 或 tickInterval<=0 视作零等待，直接返回 true（保持调用点写法简单，无需分支）。
func interruptibleSleep(d, tickInterval time.Duration, keepGoing func() bool) bool {
	if d <= 0 {
		return true
	}
	if tickInterval <= 0 || tickInterval > d {
		tickInterval = d
	}
	deadline := time.Now().Add(d)
	for {
		remain := time.Until(deadline)
		if remain <= 0 {
			return true
		}
		if remain > tickInterval {
			remain = tickInterval
		}
		time.Sleep(remain)
		if !keepGoing() {
			return false
		}
	}
}
