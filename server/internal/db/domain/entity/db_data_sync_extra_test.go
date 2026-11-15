package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCountWriteSplitExtra 写入前冲突键探测的 Extra 口径：
// 默认开启（不占位）；显式关闭才落 key；改回开启时清 key（避免「配置了但等于默认」的噪声）。
func TestCountWriteSplitExtra(t *testing.T) {
	task := &DataSyncTask{}
	assert.True(t, task.ShouldCountWriteSplit(), "缺失即默认开启：存量任务与不传该字段的调用方口径不变")

	task.SetCountWriteSplit(false)
	assert.False(t, task.ShouldCountWriteSplit())
	value, ok := task.Extra[ExtraKeyCountWriteSplit]
	assert.True(t, ok, "显式关闭必须落 Extra 才有可观测性")
	assert.Equal(t, false, value, "落库值必须是 false 而不是字符串/数字，否则 JSON 回读后类型对不上")

	task.SetCountWriteSplit(true)
	assert.True(t, task.ShouldCountWriteSplit())
	_, ok = task.Extra[ExtraKeyCountWriteSplit]
	assert.False(t, ok, "显式开启等于默认，不应占位")
}
