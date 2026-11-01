package init

import (
	"os"
	"strings"
	"testing"
)

// TestConfigHotReloadHooksRegistered 守卫：模型 / Agent 配置必须注册热生效钩子
//
// 缺陷类：钩子注册被删或配置键改名后，保存配置仍然提示成功、编译与门禁全绿，
// 只有真机改完地址发现「下一轮还连旧 LLM」——这是本功能最隐蔽的失效形态。
func TestConfigHotReloadHooksRegistered(t *testing.T) {
	raw, err := os.ReadFile("init.go")
	if err != nil {
		t.Fatalf("read init.go: %v", err)
	}
	text := stripComments(string(raw))

	for _, want := range []string{
		"RegisterConfigChangeHook(aiconfig.ConfigKeyModel",
		"ReloadModelConfig()",
		"RegisterConfigChangeHook(aiconfig.ConfigKeyAgent",
		"ReloadAgentConfig()",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("init.go 缺少热生效接线 %q：保存模型配置后不会刷新运行时", want)
		}
	}
}

// stripComments 去掉行注释（说明注释里会引用这些符号名）
func stripComments(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
