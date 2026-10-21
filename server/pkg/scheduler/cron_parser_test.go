package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAddFun_CronSpecCompatibility 固定 SecondOptional 解析器的兼容行为：
// 5 字段（标准 cron）、6 字段（含秒）与描述符表达式均应能成功 AddFun，
// 非法表达式应返回 error，避免后续误改解析器导致某种格式静默失效。
func TestAddFun_CronSpecCompatibility(t *testing.T) {
	valid := []struct {
		name string
		spec string
	}{
		{"5字段标准cron", "0 0 * * *"},
		{"6字段含秒", "0 0 3 * * ?"},
		{"描述符every", "@every 1m"},
		{"描述符hourly", "@hourly"},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			id, err := AddFun(tc.spec, func() {})
			assert.NoError(t, err, "spec %q 应解析成功", tc.spec)
			assert.NotZero(t, id)
			GetCron().Remove(id)
		})
	}

	invalid := []struct {
		name string
		spec string
	}{
		{"完全非法", "not-a-cron"},
		{"字段过多", "0 0 0 * * ? 2026 extra"},
		{"字段过少", "0 0 *"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			id, err := AddFun(tc.spec, func() {})
			assert.Error(t, err, "spec %q 应返回 error", tc.spec)
			assert.Zero(t, id)
		})
	}
}

// TestValidateSpec 校验保存入口的前置闸门：不合法的表达式必须在此拦下，
// 否则只会在 AddFun 时失败并落服务端日志，用户看到「保存成功但任务永不执行」。
func TestValidateSpec(t *testing.T) {
	valid := []string{
		"0 0 * * *",                // 5 字段标准 cron
		"0 0 3 * * ?",              // 6 字段含秒
		"0 0/5 * * * ?",            // 步长
		"0 15,45 8-18 * * mon-fri", // 列表与名称别名
		"@every 1m",
		"@daily",
	}
	for _, spec := range valid {
		assert.NoError(t, ValidateSpec(spec), "spec %q 应校验通过", spec)
	}

	invalid := []string{
		"",                 // 被调度的任务没有表达式即永不触发
		"   ",              // 仅空白
		"0 0 0",            // 段数不足
		"0 0 0 * * ? 2026", // 后端未启用 Year 字段
		"0 0 0 L * ?",      // Quartz 专有写法
		"0 0 0 ? * 1#2",    // 同上
		"0 0 0 32 * ?",     // 日越界
		"0 0 0 * 0 ?",      // 月越界
		"0 0 0 ? * 7",      // 周越界（后端不做 7→0 归一）
		"0 0 22-2 * * ?",   // 区间不回绕
		"0 0/0 * * * ?",    // 步长非正
		"not-a-cron",
	}
	for _, spec := range invalid {
		assert.Error(t, ValidateSpec(spec), "spec %q 应校验失败", spec)
	}
}

// TestValidateSpec_ConsistentWithAddFun 钉住「校验与注册同源」：
// 两者必须对同一表达式给出相同结论，否则会出现校验放行、注册失败的静默失效。
func TestValidateSpec_ConsistentWithAddFun(t *testing.T) {
	for _, spec := range append([]string{"0 0 * * *", "0 0 3 * * ?", "@every 1m", "@hourly"},
		[]string{"", "0 0 0", "0 0 0 L * ?", "0 0 0 32 * ?", "0 0 22-2 * * ?", "bad"}...) {
		id, addErr := AddFun(spec, func() {})
		if addErr == nil {
			GetCron().Remove(id)
		}
		assert.Equal(t, addErr == nil, ValidateSpec(spec) == nil, "spec %q 的校验与注册结论不一致", spec)
	}
}

// TestNextFireTime 钉住 misfire 判定依赖的「下一次触发时间」计算：
// 必须与注册同源（复用 specParser），且严格返回 from 之后的下一个触发点。
func TestNextFireTime(t *testing.T) {
	// 每日 00:00：从某日 06:00 起，下一次触发应为次日 00:00
	from := time.Date(2025, 6, 15, 6, 0, 0, 0, time.UTC)
	next, err := NextFireTime("0 0 * * *", from)
	assert.NoError(t, err)
	assert.True(t, next.Equal(time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC)),
		"每日 00:00 从 06:00 起的下一次触发应为次日 00:00，实际 %s", next)
	assert.True(t, next.After(from), "下一次触发必须严格晚于 from")

	// 年任务（1 月 1 日 00:00）：从年中起，下一次触发应为次年元旦
	next2, err := NextFireTime("0 0 1 1 *", from)
	assert.NoError(t, err)
	assert.True(t, next2.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		"年任务从年中起的下一次触发应为次年元旦，实际 %s", next2)

	// 空/非法表达式返回错误（与 ValidateSpec 同源）
	_, err = NextFireTime("", from)
	assert.Error(t, err, "空表达式应返回错误")
	_, err = NextFireTime("not-a-cron", from)
	assert.Error(t, err, "非法表达式应返回错误")
}
