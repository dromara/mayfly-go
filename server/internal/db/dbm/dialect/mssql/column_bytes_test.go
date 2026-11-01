package mssql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSQLValueBytesNonHexWritesLiteralBytes 非 hex 文本按字面字节写入（与 dbi 基础实现及参数路径统一）。
//
// 历史形态：非 hex 退回普通字符串字面量，MySQL 隐式转换（等于按字节写）、PG 直接报错 ——
// 同一任务换个冲突策略就在"脏写"与"失败"之间漂移。空串仍保持字符串字面量（空 hex 与空串不可区分）。
func TestSQLValueBytesNonHexWritesLiteralBytes(t *testing.T) {
	require.Equal(t, `0x6e6f742d686578`, mssqlSQLValueBytes("not-hex"))
	require.Equal(t, "", "")
	require.Equal(t, "NULL", mssqlSQLValueBytes(nil))
}
