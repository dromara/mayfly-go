package mysql

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDbNamesTemplateExcludesAllSystemSchemas 固化「库名枚举不得下发系统库」的契约。
//
// sys 是 MySQL 5.7+ 随实例自带、由 mysql_upgrade 维护的系统库，一旦漏排除就会作为普通库
// 出现在前端库资源树里（用户点开看到的全是 sys_config、sys 内部视图）。
// 断言按集合比对而非字符串 Contains：删项、拼错大小写、改写 NOT IN 都会失败，
// 避免「模板里恰好出现 sys 字样」这类假通过。
func TestDbNamesTemplateExcludesAllSystemSchemas(t *testing.T) {
	body := metaSQL.Get(MYSQL_DBS)
	require.NotEmpty(t, body, "MYSQL_DBS 模板未被解析")

	m := regexp.MustCompile(`(?is)NOT\s+IN\s*\(([^)]*)\)`).FindStringSubmatch(body)
	require.Len(t, m, 2, "MYSQL_DBS 必须保留 NOT IN 系统库排除清单")

	excluded := make(map[string]bool)
	for _, raw := range strings.Split(m[1], ",") {
		excluded[strings.Trim(strings.TrimSpace(raw), "'\"`")] = true
	}
	for _, sys := range []string{"mysql", "information_schema", "performance_schema", "sys"} {
		assert.True(t, excluded[sys], "系统库 %q 必须被 MYSQL_DBS 排除，否则会出现在前端库资源树", sys)
	}
}

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
