package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 不连库校验 meta.sql 新增段落被正确解析且查询就位（真实结果校验留给连库 itest / 本地验证）
func TestPgsqlMetaSQLNavigatorTemplates(t *testing.T) {
	views := metaSQL.Get(PGSQL_VIEWS_KEY)
	assert.NotEmpty(t, views, "PGSQL_VIEWS 未被解析")
	assert.Contains(t, views, "relkind = 'v'")
	assert.Contains(t, views, "pg_get_viewdef")
	// schema 作为参数字面量下推（QuoteEscape），空值回退 current_schema()
	assert.Contains(t, views, "COALESCE(NULLIF('%s', ''), current_schema())")

	seqs := metaSQL.Get(PGSQL_SEQUENCES_KEY)
	assert.NotEmpty(t, seqs, "PGSQL_SEQUENCES 未被解析")
	assert.Contains(t, seqs, "relkind = 'S'")

	ddl := metaSQL.Get(PGSQL_VIEW_DDL_KEY)
	assert.NotEmpty(t, ddl, "PGSQL_VIEW_DDL 未被解析")
	assert.Contains(t, ddl, "pg_get_viewdef")
	// 视图 DDL 含 schema 与视图名两个占位符（ObjectDDL 依次以 QuoteEscape 填入）
	assert.Contains(t, ddl, "COALESCE(NULLIF('%s', ''), current_schema())")
	assert.Contains(t, ddl, `c.relname = '%s'`)
}
