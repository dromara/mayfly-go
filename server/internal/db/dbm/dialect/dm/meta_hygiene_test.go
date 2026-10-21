package dm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemasTemplateExcludesBuiltinSchemas 固化「schema 枚举不得下发内置账号」的契约。
//
// 达梦初始化时创建 SYS/SYSDBA/SYSAUDITOR/SYSSSO 四个内置账号，各自带同名默认模式；
// 不过滤会作为业务 schema 出现在前端 schema 树里，点开只能看到系统字典对象。
// 内置名固定且用户不能与其重名，排除不会误伤自建模式。
func TestSchemasTemplateExcludesBuiltinSchemas(t *testing.T) {
	body := metaSQL.Get(DM_DB_SCHEMAS)
	require.NotEmpty(t, body, "DM_DB_SCHEMAS 模板未被解析")

	upper := strings.ToUpper(body)
	for _, sys := range []string{"SYS", "SYSDBA", "SYSAUDITOR", "SYSSSO"} {
		assert.Contains(t, upper, "'"+sys+"'", "内置账号 %q 必须被 schema 枚举排除", sys)
	}
}
