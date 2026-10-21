package application

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestImportZipWritesBeforeDeleting 技能导入必须先写新内容、再清理旧文件，且落库 allowed-tools。
//
// 缺陷类（真实发生过，两处）：
//  1. 旧实现 `DeleteBySkillId` 后逐文件写，中途失败会留下「技能行在、SKILL.md 已没」的半成品，
//     ListPublishedSkills 静默跳过它（技能凭空消失且无报错）；
//  2. `ParseFrontmatter` 不读 allowed-tools、导入也不写实体，zip 导入的技能丢掉预批准工具列表
//     （每轮调用仍要审批），读写两侧都不报错。
//
// 判据是结构性的：ImportZip 体内不得再出现「先全删」，必须出现新顺序的三个调用与 allowed_tools 落库。
// 行注释先剥掉——说明注释会引用旧写法
func TestImportZipWritesBeforeDeleting(t *testing.T) {
	raw, err := os.ReadFile("skill_plugin.go")
	require.NoError(t, err)
	text := stripLineCommentsForGuard(string(raw))

	body := regexp.MustCompile(`(?s)(func \(a \*skillPluginAppImpl\) ImportZip\(.*?\n\})`).FindStringSubmatch(text)
	require.Len(t, body, 2, "找不到 ImportZip 函数体：函数被改名/移走时本守卫须同步更新")
	fn := body[1]

	require.NotContains(t, fn, "DeleteBySkillId",
		"不得先全删资源：中途失败会让 SKILL.md 消失，技能变为不可用且无报错")
	require.Contains(t, fn, "SelectBySkillId", "清理旧文件前必须先列出当前资源集")
	require.Contains(t, fn, "DeleteById", "旧版本遗留的文件要按路径差集逐个删除")
	require.Contains(t, fn, "allowed_tools", "更新分支必须把 allowed-tools 落库（roundtrip 对称）")
	require.Contains(t, fn, "AllowedTools: fmAllowedTools", "新建分支也必须写入 allowed-tools")
}

// stripLineCommentsForGuard 去掉行注释（说明注释会引用被判据排斥的旧写法）
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
