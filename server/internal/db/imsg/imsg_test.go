package imsg

import (
	"regexp"
	"sort"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"

	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

// TestI18nMsgMapsConsistent 消息ID与中英文文案必须齐备一致
//
// 导入/执行链路的失败提示直接决定运维如何处理残留数据（例如「已提交部分不会回滚」），
// 某一语言缺失文案时会静默回落成另一种语言，运维可能读错结论；
// 占位符不一致则渲染出 {{.err}} 字面量，真实错误信息丢失。
func TestI18nMsgMapsConsistent(t *testing.T) {
	zhKeys := mapKeys(Zh_CN)
	enKeys := mapKeys(En)
	assert.ElementsMatch(t, zhKeys, enKeys, "中英文消息键集合不一致（缺失语言会导致提示静默回落成另一种语言）")

	// 键集合必须是从本包起始ID开始的连续区间：imsg 用 iota 顺序编号，跳号说明有消息ID漏配文案
	sort.Slice(zhKeys, func(i, j int) bool { return zhKeys[i] < zhKeys[j] })
	for i, id := range zhKeys {
		assert.Equalf(t, i18n.MsgId(consts.ImsgNumDb+i), id, "消息ID不连续，可能存在漏配文案的消息")
	}

	pattern := regexp.MustCompile(`\{\{\s*\.([A-Za-z0-9_]+)\s*\}\}`)
	for _, id := range zhKeys {
		zhMsg, enMsg := Zh_CN[id], En[id]
		assert.NotEmptyf(t, zhMsg, "消息ID %d 中文文案为空", id)
		assert.NotEmptyf(t, enMsg, "消息ID %d 英文文案为空", id)
		// 英文文案不应残留中文（漏翻译检测）
		assert.Falsef(t, containsHan(enMsg), "消息ID %d 英文文案含中文：%s", id, enMsg)
		assert.Equalf(t, placeholders(pattern, zhMsg), placeholders(pattern, enMsg), "消息ID %d 中英文占位符不一致", id)
	}
}

func mapKeys(m map[i18n.MsgId]string) []i18n.MsgId {
	keys := make([]i18n.MsgId, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func placeholders(re *regexp.Regexp, msg string) []string {
	matches := re.FindAllStringSubmatch(msg, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	sort.Strings(names)
	return names
}

func containsHan(msg string) bool {
	for _, r := range msg {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
