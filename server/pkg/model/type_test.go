package model

import (
	"testing"

	"mayfly-go/pkg/utils/collx"

	"github.com/stretchr/testify/require"
)

// SetExtraValue 的零值语义是「删键」，不是「跳过」。
//
// 这条用例钉住三件容易被改回去的事：① 传空串/0/false 必须把已有键清掉（旧语义是静默保留旧值，
// 表现为「把备注改回空」「关成 false」保存成功却不生效）；② 容器为 nil 时不得为了空删除而凭空
// 建出 `{}`（会多落一列无意义 JSON）；③ 缺键与零值在读取侧必须等价，否则删键会改变调用方读到的值
func TestExtraDataZeroValueRemovesKey(t *testing.T) {
	var e ExtraData

	// nil 容器 + 零值：不该建容器（落库为 NULL 而不是 "{}"）
	e.SetExtraValue("remark", "")
	require.Nil(t, e.Extra, "空容器 + 空值不得建出 map")

	e.SetExtraValue("remark", "内置模板")
	require.Equal(t, "内置模板", e.GetExtraString("remark"))

	// 空串 = 清空：键必须真的没了，而不只是读回空串
	e.SetExtraValue("remark", "")
	require.NotContains(t, e.Extra, "remark", "传空串必须删键")
	require.Empty(t, e.GetExtraString("remark"))

	// 数值/布尔零值同理：关开关与配置改回 0 都是合法编辑
	e.SetExtraValue("dryRun", true)
	e.SetExtraValue("sleepMs", 200)
	require.True(t, e.GetExtraBool("dryRun"))
	require.Equal(t, 200, e.GetExtraInt("sleepMs"))

	e.SetExtraValue("dryRun", false)
	e.SetExtraValue("sleepMs", 0)
	require.NotContains(t, e.Extra, "dryRun", "false 必须删键")
	require.NotContains(t, e.Extra, "sleepMs", "0 必须删键")
	require.False(t, e.GetExtraBool("dryRun"), "缺键读回零值，与删键前语义一致")
	require.Zero(t, e.GetExtraInt("sleepMs"))

	// 只动目标键：删一个键不得牵连同容器其他键（Extra 是一整列 JSON，多键共存是常态）
	e2 := ExtraData{Extra: collx.M{"remark": "x", "ownerUsername": "ops"}}
	e2.SetExtraValue("remark", "")
	require.Equal(t, "ops", e2.GetExtraString("ownerUsername"), "删 remark 不得抹掉其他键")

	// 链式返回自身（含零值分支），调用方可以连续写
	e3 := (&ExtraData{}).SetExtraValue("a", "").SetExtraValue("b", "v")
	require.Equal(t, "v", e3.GetExtraString("b"))
}

// 存在性判断与零值读取的一致性：靠 ok 判断是否配置的调用方，删键后必须读到「未配置」
func TestExtraDataPresenceConsistency(t *testing.T) {
	e := ExtraData{Extra: collx.M{"skipValidation": true}}
	require.True(t, func() bool { v, ok := e.Extra["skipValidation"]; return ok && v == true }())

	e.SetExtraValue("skipValidation", false)
	_, ok := e.Extra["skipValidation"]
	require.False(t, ok, "写 false 后键应消失（等价于未配置）")
	require.False(t, e.GetExtraBool("skipValidation"))
}
