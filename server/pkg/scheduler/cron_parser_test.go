package scheduler

import (
	"testing"

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
