package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSendMsgCallersAreWhitelisted 模板发送的下沉口 `SendMsg` 只能被已登记的几条发送路径调用。
//
// 为什么需要这条守卫：模板"可发性"（已启用 + 已关联渠道）只有 `Send(code)` 这条路径在**发送时**校验；
// 另外两条各有不同的责任面，都是有意的：
//   - 告警通知器：模板可按内置默认回退、渠道由通知策略路由，套用"必须绑定渠道"会拦掉正常的告警；
//   - EventBus 待办通知：在**保存期**（SaveBizTmpl）校验可发性，发送时只认事件里带来的模板。
//
// 因此真正要防的不是"现在少校验"，而是**将来新增第四条发送路径时悄悄绕过判据**：`SendMsg` 直接收
// 模板与渠道，谁都能调，编译与现有用例都不会红。这里把三个已知调用点钉成白名单——新增调用点必须
// 显式改本用例，等于强制作者回答一句「这条路径的可发性由谁负责」
func TestSendMsgCallersAreWhitelisted(t *testing.T) {
	// 本测试的工作目录是 internal/msg/application：向上一层是 internal/msg，再上一层即 internal/
	internalRoot := filepath.Join("..", "..")
	allowed := map[string]string{
		"msg/application/msg_tmpl.go":            "定义处 + Send(code) 这条已校验的路径",
		"alert/infra/notifier/alert_notifier.go": "告警通知器：渠道由通知策略路由、模板可回退默认",
		"msg/init/init.go":                       "EventBus 待办通知：保存期已校验",
	}

	found := map[string]bool{}
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := stripLineCommentsForGuard(string(raw))
		if !strings.Contains(text, ".SendMsg(") {
			return nil
		}
		rel, relErr := filepath.Rel(internalRoot, path)
		if relErr != nil {
			return relErr
		}
		found[filepath.ToSlash(rel)] = true
		return nil
	})
	require.NoError(t, err)

	// 防空跑：三条已知路径必须都扫到，否则说明扫描根或判据失效
	require.GreaterOrEqual(t, len(found), len(allowed),
		"只扫到 %d 个 SendMsg 调用点，少于已知的三条路径 —— 扫描根或判据失效", len(found))

	for file := range found {
		_, ok := allowed[file]
		require.Truef(t, ok,
			"%s 新增了对 SendMsg 的调用：请确认这条发送路径的「模板可发性」由谁负责，再把它加进本就例的白名单与说明",
			file)
	}
}

// stripLineCommentsForGuard 去掉行注释：说明注释里会原样写出 `.SendMsg(` 作为反面例子。
//
// 前提：不识别字符串字面量内的 `//`；被扫文件若出现此类字面量须改判据，而不是放宽断言
func stripLineCommentsForGuard(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
