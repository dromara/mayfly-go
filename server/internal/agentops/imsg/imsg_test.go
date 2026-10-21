package imsg

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"

	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestI18nMsgMapsConsistent 消息 id 与中英文文案必须齐备一致。
//
// 本模块的校验错误会以 toast 直接呈现给运维人员：某一语言漏配时静默回退成另一种语言，
// 中文界面上就冒出英文串（本轮真机就复现过「agentops kill switch engaged…」直出）；
// 占位符不一致则渲染出 {{.err}} 字面量，被拦的人既看不懂也定位不到是哪条配置。
//
// id 连续性判据同时钉住「不要在中间插入消息」：iota 平移会让已写进 t_sys_log 的历史
// 操作日志集体错译。
func TestI18nMsgMapsConsistent(t *testing.T) {
	zhKeys := mapKeys(Zh_CN)
	enKeys := mapKeys(En)
	assert.ElementsMatch(t, zhKeys, enKeys, "中英文消息键集合不一致")

	sort.Slice(zhKeys, func(i, j int) bool { return zhKeys[i] < zhKeys[j] })
	for index, id := range zhKeys {
		assert.Equalf(t, i18n.MsgId(consts.ImsgNumAgentops+index), id, "消息ID不连续，可能存在漏配文案的消息")
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

// TestNoHardcodedBizMessageLeft 本模块不得再有硬编码文案的业务错误。
//
// 判据落在源码而非行为上：新增一条 errorx.NewBiz("英文字面量") 编译、测试、界面都不会红，
// 只有中文界面会静默混入英文。此处把「必须走 imsg」变成会红的门禁（NewBizI / 透传既有 error
// 的 NewBiz(err.Error()) 不算硬编码）。
func TestNoHardcodedBizMessageLeft(t *testing.T) {
	root := ".."
	var violations []string
	// 匹配带字符串字面量首参的 NewBiz/NewBizf；errorx.NewBiz(err.Error()) 属于透传既有 error，不在此列
	hardcoded := regexp.MustCompile(`errorx\.NewBiz(f)?\(\s*"`)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.Contains(path, "/imsg/") {
			return nil // 文案本体就在 imsg 里，不参与本判据
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil // 用例里的断言文本不是对外文案
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		for i, line := range strings.Split(string(data), "\n") {
			if hardcoded.MatchString(line) {
				violations = append(violations, fmt.Sprintf("%s:%d  %s", path, i+1, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	require.NoError(t, err)
	assert.Emptyf(t, violations, "以下位置仍在硬编码业务文案，须改走 imsg msgId：\n%s", strings.Join(violations, "\n"))
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
