package application

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"mayfly-go/internal/flow/imsg"

	"github.com/stretchr/testify/require"
)

// TestNoHardcodedBizErrorLiterals flow 目录禁止以英文句子作为业务错误首参。
//
// 缺陷类（真实发生过）：17 处 `errorx.NewBiz("procdef not found")` 之类直透前端 toast，
// 中文界面冒出英文；其中两处还会写进审批意见（AI 审核失败原因）。
// 例外只有两种：imsg 目录本身，以及 `errorx.NewBiz(err.Error())` 这类透传（首参不是字面量）。
//
// 判据用正则匹配「首参是字符串字面量」，注释先剥掉（注释会引用被禁写法）
func TestNoHardcodedBizErrorLiterals(t *testing.T) {
	literalFirstArg := regexp.MustCompile(`errorx\.NewBizf?\(\s*"`)
	root := ".." // internal/flow

	var offenders []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if filepath.Base(path) == "imsg" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := stripLineCommentsForFlowGuard(string(raw))
		if literalFirstArg.MatchString(text) {
			rel, _ := filepath.Rel(root, path)
			offenders = append(offenders, filepath.ToSlash(rel))
		}
		return nil
	})
	require.NoError(t, err)
	require.Emptyf(t, offenders, "以下文件仍有硬编码业务错误文案（应走 imsg）：%v", offenders)
}

// TestImsgKeyParity 中英文键与占位符必须成对（漏配只在切语言时暴露）
func TestImsgKeyParity(t *testing.T) {
	placeholder := regexp.MustCompile(`\{\{\.[a-zA-Z0-9_]+\}\}`)
	require.NotEmpty(t, imsg.Zh_CN)
	require.Len(t, imsg.En, len(imsg.Zh_CN), "中英文键集合数量不一致")
	for id, zh := range imsg.Zh_CN {
		en, ok := imsg.En[id]
		require.Truef(t, ok, "英文文案缺失：msgId=%d", id)
		require.NotEmptyf(t, zh, "中文文案为空：msgId=%d", id)
		require.NotEmptyf(t, en, "英文文案为空：msgId=%d", id)
		require.ElementsMatchf(t, placeholder.FindAllString(zh, -1), placeholder.FindAllString(en, -1),
			"中英占位符不一致：msgId=%d", id)
	}
}
