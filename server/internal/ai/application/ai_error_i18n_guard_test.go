package application

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// userFacingErrorFiles 面向**普通用户**的错误出口：这些文件里的业务错误必须是 imsg 文案，
// 不得出现硬编码英文句子（会经 biz.ErrIsNil 直透前端 toast）。
//
// 管理员侧的插件/技能校验串（plugin_instance.go / skill_plugin.go / plugin_type.go）
// 仍是英文：它们只在配置插件/导入 zip 时出现，受众是管理员，且量大；
// 收口它们需要另开一轮批量改造（改动面比这里大一个数量级），先作为已知债务显式记录，
// 不得把本清单当成「全仓已无英文错误」的证明
var userFacingErrorFiles = []string{
	"conversation.go",
	"agent_runner.go",
	"agent_profile.go",
	"../api/plugin.go",
	"../tools/monitor/monitor.go",
}

// TestUserFacingErrorsAreI18n 用户可见的错误文案必须走 imsg。
//
// 缺陷类（真实发生过）：会话授权拒绝、headless 入口拒绝等错误以英文句子硬编码，
// 中文界面上弹出的是 "no permission to read this conversation"。
func TestUserFacingErrorsAreI18n(t *testing.T) {
	for _, file := range userFacingErrorFiles {
		raw, err := os.ReadFile(file)
		require.NoErrorf(t, err, "读取 %s 失败（路径或文件名变更时本守卫须同步更新）", file)
		text := stripCommentLines(string(raw))
		require.NotContainsf(t, text, `errorx.NewBiz("`,
			"%s 仍有硬编码英文业务错误：改用 errorx.NewBizI(ctx, imsg.<Key>)", file)
		require.NotContainsf(t, text, `errorx.NewBizf("`,
			"%s 仍有硬编码英文业务错误（带参数）：同样应走 imsg 文案 + 占位符", file)
	}
}

// stripCommentLines 去掉行注释（说明注释会引用被禁的写法）
func stripCommentLines(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
