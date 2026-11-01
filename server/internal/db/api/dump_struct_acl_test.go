package api

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStructOnlyDumpIsAlsoGoverned 只导表结构同样要过表/列权限。
//
// 历史缺陷：权限判定写在 `if needData` 里，`DumpType=1`（只要结构）时整段跳过，
// 受保护表的表名/列名/列类型/默认值/注释可被完整取走。权限是硬边界，不因导出形态而异。
func TestStructOnlyDumpIsAlsoGoverned(t *testing.T) {
	content, err := os.ReadFile("db.go")
	require.NoError(t, err)
	source := string(content)

	// 判定收口到 sqlexec.AssertDumpTablesAllowed（单点），且必须出现在 needData 分支之前
	aclIdx := strings.Index(source, "sqlexec.AssertDumpTablesAllowed(")
	needDataIdx := strings.Index(source, "if needData {")
	require.Greater(t, aclIdx, -1, "未找到导出整表权限判定")
	require.Greater(t, needDataIdx, -1, "未找到 needData 分支")
	require.Less(t, aclIdx, needDataIdx, "表/列权限判定必须在 needData 分支之前，覆盖仅结构导出")
}
