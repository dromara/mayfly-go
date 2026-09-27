package keyvalue

import (
	"strings"
	"testing"
)

// TestConsoleHintsContract 命令控制台的快捷命令模板契约：
// 模板要么是无参命令，要么必须带 {key} 占位符——前端靠它把当前 key 名替换进去，
// 漏写占位符会让命令指向一个不存在的字面 key，新增视角时在这里拦住
func TestConsoleHintsContract(t *testing.T) {
	views := 0
	for _, desc := range AllDescriptors() {
		if len(desc.ConsoleHints) == 0 {
			t.Fatalf("view [%s] must declare console hints for the command console", desc.View)
		}
		views++
		for _, hint := range desc.ConsoleHints {
			fields := strings.Fields(hint)
			if len(fields) == 0 {
				t.Fatalf("view [%s] has an empty console hint", desc.View)
			}
			if len(fields) == 1 {
				continue
			}
			if !strings.Contains(hint, consoleKeyPlaceholder) {
				t.Fatalf("view [%s] hint [%s] must contain the %s placeholder", desc.View, hint, consoleKeyPlaceholder)
			}
		}
	}
	if views == 0 {
		t.Fatal("no view descriptors registered")
	}
}
