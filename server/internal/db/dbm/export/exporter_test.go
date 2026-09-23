package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// 导出列映射解析（resolveDumpColumnKeys）单元测试：不依赖真实数据库。
//
// 该函数是「导出是否丢数据」的最后一道闸门：dump 按**元数据列顺序**组装 INSERT 的 VALUES，
// 而取值用的是查询结果 row 的 key。后端返回的列标签与元数据列名可能存在大小写差异
// （如部分后端统一转大写），此时按列名直接索引不命中会静默得到 nil 并导出 NULL——
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

// TestResolveDumpColumnKeysCaseInsensitive 后端将列标签统一转为大写时（oracle/mssql 常见），
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

// newConvExporter 构造仅用于测试标识符转换/开关行为的编排器。
// 传入非 nil settings 以绕开 NewExporter 内 consumer.Format() 的 nil 解引用（这些用例不需要 consumer）。
func newConvExporter() *Exporter {
	return NewExporter(nil, nil, &Settings{}, nil)
}

func TestValidateNameConversion(t *testing.T) {
	require.Error(t, ValidateNameConversion([]string{"Orders", "orders"}, nil, NameCaseLower, "postgres"))
	require.Error(t, ValidateNameConversion([]string{"t"}, col("t", "ID", "id"), NameCaseUpper, "postgres"))
	generated := dbi.Column{TableName: "t", ColumnName: "Total"}
	dbi.MarkGeneratedColumn(&generated, "postgres", `("Amount" * 2)`, dbi.GenerationStored)
	require.Error(t, ValidateNameConversion([]string{"t"}, []dbi.Column{generated}, NameCaseUpper, "postgres"))
	require.NoError(t, ValidateNameConversion([]string{"t"}, []dbi.Column{generated}, NameCaseNone, "postgres"))
	require.NoError(t, ValidateNameConversion([]string{"t"}, []dbi.Column{generated}, NameCaseUpper, "mysql"))
}

// TestExporterDefaults 新建编排器默认建表前 DROP、不做大小写转换（兼容既有硬编码行为，
// 避免零值反转导致既有备份/迁移静默改变语义）
func TestExporterDefaults(t *testing.T) {
	e := newConvExporter()
	assert.True(t, e.dropBeforeCreate, "默认必须建表前 DROP")
	assert.Equal(t, NameCaseNone, e.nameCase, "默认不转换大小写")
}

// TestWithSkipDropTable deleteTable=否 → SkipDropTable=true → 不生成 DROP；
// 反之保持 DROP。这是「保留已有表」能力的开关，语义反转即数据被误删或建表失败
func TestWithSkipDropTable(t *testing.T) {
	e := newConvExporter()
	assert.True(t, e.WithSkipDropTable(false).dropBeforeCreate, "不跳过时仍应 DROP")
	assert.False(t, e.WithSkipDropTable(true).dropBeforeCreate, "跳过时不得 DROP")
}

// TestConvIdent 大小写转换策略：None/0 原样、Upper 转大写、Lower 转小写
func TestConvIdent(t *testing.T) {
	assert.Equal(t, "T_Order", newConvExporter().WithNameCase(NameCaseNone).convIdent("T_Order"))
	assert.Equal(t, "T_Order", newConvExporter().WithNameCase(0).convIdent("T_Order"), "0 按不转换处理")
	assert.Equal(t, "T_ORDER", newConvExporter().WithNameCase(NameCaseUpper).convIdent("T_Order"))
	assert.Equal(t, "t_order", newConvExporter().WithNameCase(NameCaseLower).convIdent("T_Order"))
}

// TestConvColumnsDoesNotMutateInput 转换必须返回副本，不得就地修改入参：
// 源库查询与列映射（resolveDumpColumnKeys）仍依赖原始列名，若被就地改写将导致取值错位/NULL
func TestConvColumnsDoesNotMutateInput(t *testing.T) {
	src := col("t_order", "id", "Amount")
	e := newConvExporter().WithNameCase(NameCaseUpper)

	got := e.convColumns(src)
	assert.Equal(t, "ID", got[0].ColumnName)
	assert.Equal(t, "AMOUNT", got[1].ColumnName)
	assert.Equal(t, "T_ORDER", got[0].TableName, "所属表名同步转换")

	// 入参保持原样
	assert.Equal(t, "id", src[0].ColumnName, "不得就地修改入参列名")
	assert.Equal(t, "Amount", src[1].ColumnName)
	assert.Equal(t, "t_order", src[0].TableName)
}

// TestConvColumnsNoneReturnsSame None/0 时直接返回原切片（零拷贝快路径）
func TestConvColumnsNoneReturnsSame(t *testing.T) {
	src := col("t_order", "id")
	e := newConvExporter().WithNameCase(NameCaseNone)
	assert.Equal(t, "id", e.convColumns(src)[0].ColumnName)
}

// TestConvTableAndIndexes 表名与索引列名按策略转换，供建表/建索引 DDL 引用转换后的标识符
func TestConvTableAndIndexes(t *testing.T) {
	e := newConvExporter().WithNameCase(NameCaseLower)

	tbl := e.convTable(dbi.Table{TableName: "T_Order"})
	assert.Equal(t, "t_order", tbl.TableName)

	idxs := e.convIndexes([]dbi.Index{{ColumnName: "ID"}, {ColumnName: "Amount"}})
	assert.Equal(t, "id", idxs[0].ColumnName)
	assert.Equal(t, "amount", idxs[1].ColumnName)
}
