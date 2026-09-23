package oracle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 不连库校验 meta.sql 新增段落被正确解析且查询就位（真实结果校验留给连库 itest / 本地验证）
func TestOracleMetaSQLNavigatorTemplates(t *testing.T) {
	views := metaSQL.Get(ORACLE_VIEWS_KEY)
	assert.NotEmpty(t, views, "ORACLE_VIEWS 未被解析")
	assert.Contains(t, views, "ALL_VIEWS")
	assert.Contains(t, views, "CURRENT_SCHEMA")
	// schema 以参数字面量下推（QuoteEscape）；oracle 空串即 NULL，COALESCE 回退当前模式
	assert.Contains(t, views, "COALESCE('%s',")

	seqs := metaSQL.Get(ORACLE_SEQUENCES_KEY)
	assert.NotEmpty(t, seqs, "ORACLE_SEQUENCES 未被解析")
	assert.Contains(t, seqs, "ALL_SEQUENCES")
	assert.Contains(t, seqs, "COALESCE('%s',")
}
