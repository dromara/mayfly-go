package imsg

import (
	"regexp"
	"sort"
	"testing"
	"unicode"

	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"

	"github.com/stretchr/testify/assert"
)

// TestI18nMsgMapsConsistent 消息 id 与中英文文案必须齐备一致。
//
// 触发策略的校验原因有几十条，会以提示直接呈现给管理员：某一语言漏配时会静默回落成
// 另一种语言，中文界面上就冒出英文串；占位符不一致则渲染出 {{.field}} 字面量，
// 被拦的人既看不懂也定位不到是哪条规则
func TestI18nMsgMapsConsistent(t *testing.T) {
	zhKeys := mapKeys(Zh_CN)
	enKeys := mapKeys(En)
	assert.ElementsMatch(t, zhKeys, enKeys, "中英文消息键集合不一致")

	// imsg 用 iota 顺序编号，键集合必须是从本模块起始 id 开始的连续区间：
	// 跳号说明有消息 id 只在一边配了文案
	sort.Slice(zhKeys, func(i, j int) bool { return zhKeys[i] < zhKeys[j] })
	for index, id := range zhKeys {
		assert.Equalf(t, i18n.MsgId(consts.ImsgNumFlow+index), id, "消息ID不连续，可能存在漏配文案的消息")
	}

	pattern := regexp.MustCompile(`\{\{\s*\.([A-Za-z0-9_]+)\s*\}\}`)
	for _, id := range zhKeys {
		zhMsg, enMsg := Zh_CN[id], En[id]
		assert.NotEmptyf(t, zhMsg, "消息ID %d 中文文案为空", id)
		assert.NotEmptyf(t, enMsg, "消息ID %d 英文文案为空", id)
		assert.Falsef(t, containsHan(enMsg), "消息ID %d 英文文案含中文：%s", id, enMsg)
		assert.Equalf(t, placeholders(pattern, zhMsg), placeholders(pattern, enMsg), "消息ID %d 中英文占位符不一致", id)
	}
}

func mapKeys(msgs map[i18n.MsgId]string) []i18n.MsgId {
	keys := make([]i18n.MsgId, 0, len(msgs))
	for key := range msgs {
		keys = append(keys, key)
	}
	return keys
}

func placeholders(re *regexp.Regexp, msg string) []string {
	matches := re.FindAllStringSubmatch(msg, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}
	sort.Strings(names)
	return names
}

func containsHan(msg string) bool {
	for _, char := range msg {
		if unicode.Is(unicode.Han, char) {
			return true
		}
	}
	return false
}
