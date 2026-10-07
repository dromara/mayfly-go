package postgres

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTableSearchTemplateIsSingleStatement 固化「表搜索下推模板不得以分号结尾」的不变式。
//
// 背景：SearchTables 会在模板尾部追加 LIMIT/OFFSET 子句下推到系统目录。pg 驱动扩展协议
// (Parse/Prepare) 不接受多语句，若模板自带尾分号则拼成「...; LIMIT $2」立即报语法错误，
// 使下推静默失败并回退全量+内存过滤——回退路径同样能跑通测试，导致「下推从未真正生效」这类
// 缺陷长期潜伏（曾实际发生：ILIKE/OFFSET 一度变死代码）。此测试把该回归钉死在模板层。
func TestTableSearchTemplateIsSingleStatement(t *testing.T) {
	body := metaSQL.Get(PGSQL_TABLE_SEARCH_KEY)
	assert.NotEmpty(t, body, "PGSQL_TABLE_SEARCH 模板应能解析出内容")

	trimmed := strings.TrimRight(body, " \t\r\n")
	assert.False(t, strings.HasSuffix(trimmed, ";"), "PGSQL_TABLE_SEARCH 模板不得以分号结尾（下推会再拼 LIMIT/OFFSET）")

	// 追加子句后整条语句仍不应含任何分号（即仍是单语句，可被扩展协议接受）
	composed := trimmed + " LIMIT $2 OFFSET $3"
	assert.NotContains(t, composed, ";", "追加 LIMIT/OFFSET 后的下推 SQL 必须是单语句")
}
