package application

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestSaveNotifiesConfigChange 守卫：两条保存分支（新建 / 更新）都必须发配置变更通知
//
// 缺陷类：只有更新分支通知、新建分支直接 return 时，「首次配置 AI 模型」不会热生效，
// 而保存界面照样提示成功——表现是「改完模型配置，下一轮仍报未配置」。
func TestSaveNotifiesConfigChange(t *testing.T) {
	raw, err := os.ReadFile("config.go")
	if err != nil {
		t.Fatalf("read config.go: %v", err)
	}
	text := stripLineComments(string(raw))

	// 捕获组必须显式包住整个函数体：FindStringSubmatch 只在有捕获组时返回下标 1
	body := regexp.MustCompile(`(?s)(func \(a \*configAppImpl\) Save\(.*?\n\})`).FindStringSubmatch(text)
	if len(body) != 2 {
		t.Fatal("找不到 configAppImpl.Save 函数体：保存逻辑被移动/改名时本守卫须同步更新")
	}

	const call = "notifyConfigChanged(ctx, config.Key)"
	if got := strings.Count(body[1], call); got != 2 {
		t.Fatalf("Save 中 %s 出现 %d 次，want 2（新建与更新分支各一次）", call, got)
	}
}

// stripLineComments 去掉行注释（说明注释里会引用函数名与调用形态）
func stripLineComments(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
