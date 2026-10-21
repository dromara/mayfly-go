package oracle

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemasTemplatesGateOracleMaintained 固化「12c+ 才引用 ORACLE_MAINTAINED」的契约。
//
// ALL_USERS.ORACLE_MAINTAINED 是 12c 新增列，11g 直接引用会 ORA-00904 让整个 schema 列表请求失败；
// 因此基础模板必须保持无该列，过滤版单独成段，由 GetSchemas 探测列存在后选用。
func TestSchemasTemplatesGateOracleMaintained(t *testing.T) {
	base := metaSQL.Get(ORACLE_DB_SCHEMAS)
	require.NotEmpty(t, base, "ORACLE_DB_SCHEMAS 模板未被解析")
	assert.NotContains(t, strings.ToUpper(base), "ORACLE_MAINTAINED", "基础模板必须兼容 11g，不得引用 12c 才有的列")

	filtered := metaSQL.Get(ORACLE_DB_SCHEMAS_MAINTAINED)
	require.NotEmpty(t, filtered, "ORACLE_DB_SCHEMAS_MAINTAINED 模板未被解析")
	assert.Contains(t, strings.ToUpper(filtered), "ORACLE_MAINTAINED = 'N'", "过滤版必须只保留非 Oracle 维护的账号")
}
