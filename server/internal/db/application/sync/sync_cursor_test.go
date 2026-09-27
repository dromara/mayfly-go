package sync

import (
	"testing"
	"time"

	"mayfly-go/internal/db/domain/entity"

	"github.com/stretchr/testify/assert"
)

// ========== resolveCursorInclusivity ==========

func TestResolveCursorInclusivity_AutoByMode(t *testing.T) {
	// Auto 下按模式给安全默认：Merge/SoftDel/HardDel → Inclusive；其余 → Exclusive
	cases := []struct {
		mode     entity.DataSyncMode
		wantIncl bool
	}{
		{entity.DataSyncModeIncrementalAppend, false},
		{entity.DataSyncModeIncrementalMerge, true},
		{entity.DataSyncModeFullRefresh, false},
		{entity.DataSyncModeIncrementalSoftDel, true},
		{entity.DataSyncModeIncrementalHardDel, true},
		{entity.DataSyncModeValidation, false},
	}
	for _, c := range cases {
		assert.Equalf(t, c.wantIncl, resolveCursorInclusivity(c.mode, entity.CursorInclusivityAuto), "Auto mode=%d", c.mode)
	}
}

func TestResolveCursorInclusivity_ExplicitOverrideWins(t *testing.T) {
	// 显式覆写优先于模式默认；合法性由 validateCursorSemantics 单独把关，本函数只解析意图
	assert.False(t, resolveCursorInclusivity(entity.DataSyncModeIncrementalMerge, entity.CursorInclusivityExclusive))
	assert.True(t, resolveCursorInclusivity(entity.DataSyncModeIncrementalAppend, entity.CursorInclusivityInclusive))
}

// ========== validateCursorSemantics ==========

func TestValidateCursorSemantics(t *testing.T) {
	cases := []struct {
		name        string
		mode        entity.DataSyncMode
		inclusivity entity.CursorInclusivity
		wantErr     bool
	}{
		{"Append Auto OK", entity.DataSyncModeIncrementalAppend, entity.CursorInclusivityAuto, false},
		{"Append Exclusive OK", entity.DataSyncModeIncrementalAppend, entity.CursorInclusivityExclusive, false},
		// 核心防御：目标不幂等时禁用 Inclusive，防重发变重复
		{"Append Inclusive rejected", entity.DataSyncModeIncrementalAppend, entity.CursorInclusivityInclusive, true},
		{"Merge Inclusive OK", entity.DataSyncModeIncrementalMerge, entity.CursorInclusivityInclusive, false},
		{"Merge Exclusive OK", entity.DataSyncModeIncrementalMerge, entity.CursorInclusivityExclusive, false},
		// FullRefresh/Validation 不拼水位条件，Inclusive 无效配置也应拒
		{"FullRefresh Inclusive rejected", entity.DataSyncModeFullRefresh, entity.CursorInclusivityInclusive, true},
		{"Validation Inclusive rejected", entity.DataSyncModeValidation, entity.CursorInclusivityInclusive, true},
		{"SoftDel Inclusive OK", entity.DataSyncModeIncrementalSoftDel, entity.CursorInclusivityInclusive, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			task := &entity.DataSyncTask{SyncMode: c.mode}
			task.SetCursorInclusivity(c.inclusivity)
			err := validateCursorSemantics(task)
			if c.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ========== interruptibleSleep ==========

func TestInterruptibleSleep_ZeroFastPath(t *testing.T) {
	start := time.Now()
	ok := interruptibleSleep(0, 100*time.Millisecond, func() bool { return true })
	assert.True(t, ok)
	assert.Less(t, time.Since(start), 50*time.Millisecond, "d<=0 必须直接返回，不进入 tick 循环")
}

func TestInterruptibleSleep_CompletesFullDuration(t *testing.T) {
	start := time.Now()
	ok := interruptibleSleep(250*time.Millisecond, 50*time.Millisecond, func() bool { return true })
	assert.True(t, ok)
	// 允许少量调度误差，至少跑满 200ms
	assert.GreaterOrEqual(t, time.Since(start), 200*time.Millisecond)
}

func TestInterruptibleSleep_AbortsOnKeepGoingFalse(t *testing.T) {
	calls := 0
	start := time.Now()
	// 1s 总时长，第 2 个 tick 起返回 false；tick 50ms 时中止，总耗时应远小于 1s
	ok := interruptibleSleep(time.Second, 50*time.Millisecond, func() bool {
		calls++
		return calls < 2
	})
	assert.False(t, ok, "keepGoing=false 必须让 sleep 提前返回")
	assert.Less(t, time.Since(start), 500*time.Millisecond, "提前返回的耗时不应跑满 1s")
}

func TestInterruptibleSleep_TickLargerThanDurationCapsToDuration(t *testing.T) {
	// tickInterval > d 时按 d 一次性睡完，不做无意义切片
	start := time.Now()
	ok := interruptibleSleep(80*time.Millisecond, time.Second, func() bool { return true })
	assert.True(t, ok)
	assert.GreaterOrEqual(t, time.Since(start), 60*time.Millisecond)
}
