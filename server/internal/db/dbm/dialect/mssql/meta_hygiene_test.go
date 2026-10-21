package mssql

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDbNamesTemplateExcludesSystemDatabases 固化「库名枚举不得下发系统库」的契约。
//
// sys.databases 里 database_id 1~4 恒为 master/tempdb/model/msdb 且 state=0（online），
// 只按 state 过滤会把它们当作业务库显示在前端资源树。此测试锁住 ID 上界与分发库排除两个判据，
// 防止后续重写模板时退回「仅 state=0」。
func TestDbNamesTemplateExcludesSystemDatabases(t *testing.T) {
	body := metaSQL.Get(MSSQL_DBS_KEY)
	require.NotEmpty(t, body, "MSSQL_DBS 模板未被解析")

	assert.Regexp(t, regexp.MustCompile(`(?i)database_id\s*>\s*4`), body, "必须按 database_id 排除 1~4 的系统库")
	assert.Regexp(t, regexp.MustCompile(`(?i)is_distributor\s*=\s*0`), body, "必须排除复制分发系统库")
	assert.Regexp(t, regexp.MustCompile(`(?i)state\s*=\s*0`), body, "必须保留只枚举在线库的条件")
}
