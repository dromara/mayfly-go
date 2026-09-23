package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 不连库的模板解析验证：确保 meta.sql 新增段落能被 NewSQLTemplates 正确识别，
// 且关键查询片段就位（真实结果校验留给连库 itest / 本地验证）
func TestMysqlMetaSQLViewsAndRelationsTemplates(t *testing.T) {
	views := metaSQL.Get(MYSQL_VIEWS_KEY)
	assert.NotEmpty(t, views, "MYSQL_VIEWS 模板未被解析")
	assert.Contains(t, views, "information_schema.views")
	assert.Contains(t, views, "view_definition")
	// schema 作为绑定参数下推，空值回退当前库
	assert.Contains(t, views, "COALESCE(NULLIF(?, ''), DATABASE())")

	rels := metaSQL.Get(MYSQL_TABLE_RELATIONS_KEY)
	assert.NotEmpty(t, rels, "MYSQL_TABLE_RELATIONS 模板未被解析")
	assert.Contains(t, rels, "REFERENTIAL_CONSTRAINTS")
	assert.Contains(t, rels, "KEY_COLUMN_USAGE")
	// 外键查询含 schema 与表名两个占位符（GetForeignKeys 以参数化传入 schema、table）
	assert.Contains(t, rels, "k.table_schema = COALESCE(NULLIF(?, ''), DATABASE())")
	assert.Contains(t, rels, "k.table_name = ?")
}
