package sqlexec

// 结果集导出格式门禁的纯逻辑单测。
//
// 「只允许只读格式」这条约束是异步导出与同步导出共用的安全边界：脚本类格式（sql）
// 的产物是方言 DDL/DML，需要目标表语义，不能拿来导一次查询的结果集。判据与报错提示
// 同源（都走 readOnlyExportFormats），这里钉住两者不会各自漂移。

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsReadOnlyExportFormat(t *testing.T) {
	// csv/json 是非脚本类，可用于结果集导出
	assert.True(t, isReadOnlyExportFormat("csv"))
	assert.True(t, isReadOnlyExportFormat("json"))

	// sql 是脚本类：产物是方言脚本而非结果集，必须被挡在门外
	assert.False(t, isReadOnlyExportFormat("sql"), "脚本类格式不能用于结果集导出")

	// 未注册的格式一律拒绝
	assert.False(t, isReadOnlyExportFormat("xml"))
	assert.False(t, isReadOnlyExportFormat(""))
}

func TestReadOnlyExportFormats(t *testing.T) {
	formats := readOnlyExportFormats()
	assert.Contains(t, formats, "csv")
	assert.Contains(t, formats, "json")
	assert.NotContains(t, formats, "sql", "报错提示里不能把脚本类格式列为可用，否则用户按提示填了又被拒")
}
