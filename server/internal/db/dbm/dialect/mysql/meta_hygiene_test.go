package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTableSearchTemplateIsSingleStatement 固化「表搜索下推模板不得以分号结尾」的不变式。
//
// 背景：SearchTables 直接在 metaSQL.Get 结果尾部追加 " LIMIT ?"/" OFFSET ?"（mysql 无 pg 那样的
// TrimRight 防御），若 MYSQL_TABLE_SEARCH 模板带尾分号则拼成「...; LIMIT ?」多语句，驱动默认不开
// multiStatements 会报错，令下推静默回退。此测试钉死该回归，避免 mysql 侧同样潜伏「下推从未生效」。
func TestTableSearchTemplateIsSingleStatement(t *testing.T) {
	body := metaSQL.Get(MYSQL_TABLE_SEARCH_KEY)
	assert.NotEmpty(t, body, "MYSQL_TABLE_SEARCH 模板应能解析出内容")

	trimmed := strings.TrimRight(body, " \t\r\n")
	assert.False(t, strings.HasSuffix(trimmed, ";"), "MYSQL_TABLE_SEARCH 模板不得以分号结尾（下推会直接追加 LIMIT/OFFSET）")

	composed := trimmed + " LIMIT ? OFFSET ?"
	assert.NotContains(t, composed, ";", "追加 LIMIT/OFFSET 后的下推 SQL 必须是单语句")
}
