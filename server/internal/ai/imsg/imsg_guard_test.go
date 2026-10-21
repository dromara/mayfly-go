package imsg

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// placeholderRe 服务端插值占位符（`{{.name}}`，与前端 vue-i18n 的 `{name}` 不同）
var placeholderRe = regexp.MustCompile(`\{\{\.[a-zA-Z0-9_]+\}\}`)

// TestMsgKeySetsAndPlaceholdersMatch 中英文键集合与占位符必须成对。
//
// 缺陷类：新增键只加了一种语言 → 该语言下取不到文案（前端/接口回落到 msgId 数字或空串）；
// 中英占位符不一致 → 一种语言里插值丢失（用户看到原始 {{.profileId}} 或句子缺字段），
// 编译与调用点都发现不了，只在切语言时暴露
func TestMsgKeySetsAndPlaceholdersMatch(t *testing.T) {
	require.NotEmpty(t, Zh_CN, "中文文案表为空")
	require.Len(t, En, len(Zh_CN), "中英文键集合数量不一致")

	for id, zh := range Zh_CN {
		en, ok := En[id]
		require.Truef(t, ok, "英文文案缺失：msgId=%d", id)
		require.NotEmptyf(t, zh, "中文文案为空：msgId=%d", id)
		require.NotEmptyf(t, en, "英文文案为空：msgId=%d", id)
		require.ElementsMatchf(t, placeholderRe.FindAllString(zh, -1), placeholderRe.FindAllString(en, -1),
			"中英占位符不一致：msgId=%d, zh=%q, en=%q", id, zh, en)
	}
}
