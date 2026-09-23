package sync

import (
	_ "mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// colHelper 构造仅带 DataType 的 dbi.Column，用于 bindSyncValues 表形态解析
func colHelper(name, dataType string) dbi.Column {
	return dbi.Column{ColumnName: name, DataType: dataType}
}

func TestBindSyncValues_BoolStringToMySQLInt(t *testing.T) {
	// PG Bool 扫描为字符串 "true"/"false"，MySQL tinyint 目标列参数化必须落 1/0，
	// 否则严格模式拒字符串、非严格默默存 0（与文本路径的 true→1 隐式转语义分叉）。
	columns := []dbi.Column{colHelper("flag", "tinyint")}
	rows := [][]any{{"true"}, {"false"}, {"TRUE"}, {"  False  "}}
	bound, err := bindSyncValues("mysql", columns, rows)
	require.NoError(t, err)
	assert.Equal(t, int64(1), bound[0][0])
	assert.Equal(t, int64(0), bound[1][0])
	assert.Equal(t, int64(1), bound[2][0], "大小写不敏感")
	assert.Equal(t, int64(0), bound[3][0], "允许前后空白")
}

func TestBindSyncValues_GoBoolToMySQLInt(t *testing.T) {
	// ClickHouse Bool 走 DTBool，扫描得到 Go bool；同样归一为 int64 保持绑定契约一致
	columns := []dbi.Column{colHelper("flag", "tinyint")}
	bound, err := bindSyncValues("mysql", columns, [][]any{{true}, {false}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), bound[0][0])
	assert.Equal(t, int64(0), bound[1][0])
}

func TestBindSyncValues_NilAndNonBooleanPassthrough(t *testing.T) {
	// NULL 保留：nil 跳过归一，直接透传，让驱动写入目标列的 NULL
	// 非布尔文本不动：整数列收到 "42" 或 "hello" 应原样交给驱动，交由其自身解析/报错，
	// 应用层不做过度解读以免吞掉真实的数据质量问题。
	columns := []dbi.Column{colHelper("flag", "tinyint")}
	bound, err := bindSyncValues("mysql", columns, [][]any{{nil}, {"42"}, {"hello"}, {int64(7)}})
	require.NoError(t, err)
	assert.Nil(t, bound[0][0])
	assert.Equal(t, "42", bound[1][0])
	assert.Equal(t, "hello", bound[2][0])
	assert.Equal(t, int64(7), bound[3][0])
}

func TestBindSyncValues_BoolTargetKeptAsBool(t *testing.T) {
	// 目标为 PG bool（TCBool）：Go bool 无需归一为 int，直接透传（PG 驱动接受原生 bool）。
	// 只有整型/位类目标才走 bool→int 归一，避免把 bool 列硬塞成 0/1 破坏类型语义。
	columns := []dbi.Column{colHelper("flag", "bool")}
	bound, err := bindSyncValues("postgres", columns, [][]any{{true}, {false}})
	require.NoError(t, err)
	assert.Equal(t, true, bound[0][0])
	assert.Equal(t, false, bound[1][0])
}

func TestBindSyncValues_BinaryHexStillDecoded(t *testing.T) {
	// 回归：既有 hex 文本 → []byte 归一不受本次新增分支影响
	columns := []dbi.Column{colHelper("blob_col", "blob")}
	bound, err := bindSyncValues("mysql", columns, [][]any{{"48656c6c6f"}})
	require.NoError(t, err)
	assert.Equal(t, []byte("Hello"), bound[0][0])
}

func TestBindSyncValues_EmptyHexBlob(t *testing.T) {
	// ValuerBytes 对空字节集输出空串（非 nil），绑定时应归一为 len==0 的 []byte；
	// 上一版误用 IsHexString 将空串拒为非法，导致 X'' 写入的 blob 列参数化链路回归失败。
	columns := []dbi.Column{colHelper("payload", "blob")}
	bound, err := bindSyncValues("mysql", columns, [][]any{{""}})
	require.NoError(t, err)
	assert.Equal(t, []byte{}, bound[0][0])
}

func TestBindSyncValues_InvalidBinaryStillFails(t *testing.T) {
	// 非 hex 字符应报错，不默默当字符串写进 blob：
	// 奇数长度与包含非 hex 字符两种都由 hex.DecodeString 自身拒绝。
	columns := []dbi.Column{colHelper("blob_col", "blob")}
	_, err := bindSyncValues("mysql", columns, [][]any{{"zz"}})
	assert.Error(t, err)
	_, err = bindSyncValues("mysql", columns, [][]any{{"4865"}})
	assert.NoError(t, err, "偶数位合法 hex 应能通过")
}

func TestBindSyncValues_RowColumnMismatch(t *testing.T) {
	columns := []dbi.Column{colHelper("a", "int"), colHelper("b", "int")}
	_, err := bindSyncValues("mysql", columns, [][]any{{int64(1)}})
	assert.Error(t, err)
}
