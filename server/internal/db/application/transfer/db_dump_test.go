package transfer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// 导出列映射解析（resolveDumpColumnKeys）单元测试：不依赖真实数据库。
//
// 该函数是「导出是否丢数据」的最后一道闸门：dump 按**元数据列顺序**组装 INSERT 的 VALUES，
// 而取值用的是查询结果 row 的 key。驱动返回的列标签与元数据列名可能存在大小写差异
// （如部分驱动统一转大写），此时按列名直接索引不命中会静默得到 nil 并导出 NULL——
// 备份文件看起来完全正常，恢复后该列全为空，属不可逆数据丢失。
//
// 真实数据库集成测试（-tags it）只会在方言行为恰好命中时间接覆盖，CI 不带该标签时此处是唯一守卫。
func qc(names ...string) []*dbi.QueryColumn {
	cols := make([]*dbi.QueryColumn, 0, len(names))
	for _, n := range names {
		cols = append(cols, &dbi.QueryColumn{Name: n, Key: n})
	}
	return cols
}

func col(table string, names ...string) []dbi.Column {
	cols := make([]dbi.Column, 0, len(names))
	for _, n := range names {
		cols = append(cols, dbi.Column{TableName: table, ColumnName: n})
	}
	return cols
}

// TestResolveDumpColumnKeysExactMatch 列名完全一致时按元数据顺序返回同名 key
func TestResolveDumpColumnKeysExactMatch(t *testing.T) {
	keys, err := resolveDumpColumnKeys(col("t_order", "id", "amount", "remark"), qc("id", "amount", "remark"))
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "amount", "remark"}, keys)
}

// TestResolveDumpColumnKeysKeepsMetadataOrder 查询结果列顺序与元数据不同时，
// 必须按元数据顺序返回 key（VALUES 与列清单的对应关系由元数据顺序决定，错位即写入错值）
func TestResolveDumpColumnKeysKeepsMetadataOrder(t *testing.T) {
	keys, err := resolveDumpColumnKeys(col("t_order", "id", "amount", "remark"), qc("remark", "amount", "id"))
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "amount", "remark"}, keys, "返回值顺序必须跟随元数据列")
}

// TestResolveDumpColumnKeysCaseInsensitive 驱动将列标签统一转为大写时（oracle/mssql 常见），
// 需大小写不敏感命中，否则整列导出为 NULL
func TestResolveDumpColumnKeysCaseInsensitive(t *testing.T) {
	keys, err := resolveDumpColumnKeys(col("t_order", "id", "amount"), []*dbi.QueryColumn{
		{Name: "ID", Key: "ID"}, {Name: "AMOUNT", Key: "AMOUNT"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"ID", "AMOUNT"}, keys)
}

// TestResolveDumpColumnKeysPrefersExactOverFolded 结果集存在两个大小写折叠后同名的列时
// （如 SELECT id, ID 或驱动为重复标签生成的唯一 Key），必须取精确匹配的那个 Key：
// 大小写折叠索引会被后出现的列覆盖，一旦误命中就会把另一列的值写入本列（静默错值）
func TestResolveDumpColumnKeysPrefersExactOverFolded(t *testing.T) {
	keys, err := resolveDumpColumnKeys(col("t_order", "id"), []*dbi.QueryColumn{
		{Name: "id", Key: "id"}, {Name: "ID", Key: "ID_dup"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"id"}, keys, "精确匹配必须优先于大小写折叠匹配")
}

// TestResolveDumpColumnKeysMissingColumn 元数据列在结果集中不存在时必须报错终止导出：
// 静默跳过等于用 NULL 覆盖真实数据
func TestResolveDumpColumnKeysMissingColumn(t *testing.T) {
	keys, err := resolveDumpColumnKeys(col("t_order", "id", "amount", "secret"), qc("id", "amount"))
	require.Error(t, err, "列缺失时必须失败，不得返回可用于导出的 key")
	assert.Nil(t, keys)
	assert.Contains(t, err.Error(), "secret", "错误信息需指明缺失的列名")
	assert.Contains(t, err.Error(), "t_order", "错误信息需指明所属表")
}

// TestResolveDumpColumnKeysEmptyQueryColumns 结果集无列信息（元数据查询失败/驱动异常）时不得放行
func TestResolveDumpColumnKeysEmptyQueryColumns(t *testing.T) {
	_, err := resolveDumpColumnKeys(col("t_order", "id"), nil)
	require.Error(t, err)

	// 元数据无列时不产生任何 key，交由调用方（DumpDbScript 的列缺失校验）处理
	keys, err := resolveDumpColumnKeys(col("t_order"), qc("id"))
	require.NoError(t, err)
	assert.Empty(t, keys)
}

// TestResolveDumpColumnKeysErrorTextSafe 表/列名来自元数据且可含换行（恶意或异常库表），
// 错误文本必须归一化，否则日志与前端展示会被伪造行内容
func TestResolveDumpColumnKeysErrorTextSafe(t *testing.T) {
	_, err := resolveDumpColumnKeys(col("t\n-- DROP TABLE x", "missing"), qc("id"))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "\n", "错误信息不得含裸换行: %q", err.Error())
	assert.NotContains(t, err.Error(), "DROP TABLE x \n", "换行需被归一化，避免注释提前结束")
}
