//go:build it

package itest

// MySQL 删除对账文本键字节保真：反斜杠、单引号、Unicode、空串、NUL、换行等键值
// 经十六进制字面量内联后，删除语句不得因转义差异误删保留行或漏删多余行。

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestITMysqlReconcileTextKeys(t *testing.T) {
	c := mysqlConn(t)
	defer c.Close()
	mustExec(t, c, "CREATE TABLE it_reconcile_text (id VARCHAR(128) PRIMARY KEY) DEFAULT CHARSET utf8mb4")
	keys := []string{"a\\b", "quote'key", "中文😀", "", "null\x00byte", "line\nbreak"}
	retained := make([][]any, 0, len(keys))
	for _, key := range keys {
		_, err := c.Exec("INSERT INTO it_reconcile_text VALUES (?)", key)
		require.NoError(t, err)
		retained = append(retained, []any{key})
	}
	mustExec(t, c, "INSERT INTO it_reconcile_text VALUES ('remove-me')")
	sqls := c.GetDialect().GetSQLGenerator().GenBatchDelete("it_reconcile_text", []string{"id"}, retained, nil)
	require.Len(t, sqls, 1)
	affected, err := c.Exec(sqls[0])
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	_, rows, err := c.Query("SELECT id FROM it_reconcile_text")
	require.NoError(t, err)
	actual := make([]string, 0, len(rows))
	for _, row := range rows {
		actual = append(actual, fmt.Sprint(row["id"]))
	}
	assert.ElementsMatch(t, keys, actual, "保留键应逐字节保真，多余行应精确删除")
}
