package skill
package skill

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseFrontmatterReadsAllowedTools allowed-tools 必须能被解析出来。
//
// 缺陷类（真实发生过）：BuildSkillMd 会写 allowed-tools，而 ParseFrontmatter 只解析
// name/description——zip 导入的技能因此丢掉预批准工具列表：技能装上了、每轮调用仍要审批，
// 写读两侧都不报错（roundtrip 静默丢字段）。
func TestParseFrontmatterReadsAllowedTools(t *testing.T) {
	md := "---\nname: db-triage\ndescription: 数据库排障\nallowed-tools: DbQueryTables MachineCommandExec\n---\n\n正文内容\n"

	name, desc, allowed, body := ParseFrontmatter(md)
	require.Equal(t, "db-triage", name)
	require.Equal(t, "数据库排障", desc)
	require.Equal(t, "DbQueryTables MachineCommandExec", allowed)
	require.Equal(t, "正文内容", body)

	// 没有 frontmatter 时必须原样当正文返回，allowed 为空
	_, _, allowed, body = ParseFrontmatter("纯正文")
	require.Empty(t, allowed)
	require.Equal(t, "纯正文", body)
}

// TestBuildAndParseSkillMdRoundtrip 导出/导入必须 roundtrip 一致（写读同一真源）。
func TestBuildAndParseSkillMdRoundtrip(t *testing.T) {
	md := BuildSkillMd("db-triage", "数据库排障", "DbQueryTables MachineCommandExec", "正文内容")

	name, desc, allowed, body := ParseFrontmatter(md)
	require.Equal(t, "db-triage", name)
	require.Equal(t, "数据库排障", desc)
	require.Equal(t, "DbQueryTables MachineCommandExec", allowed, "allowed-tools 在 roundtrip 后丢失")
	require.Equal(t, "正文内容", body)
}
