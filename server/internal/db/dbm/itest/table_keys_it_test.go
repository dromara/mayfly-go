package itest

import (
	"context"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertKeyColumnOrder 断言：keys 中存在一个类型为 kt、列名顺序恰为 wantCols 的约束（按定义顺序，忽略约束名）。
//
// 联合主键/唯一键的「键内列顺序」必须来自约束定义顺序，而非列在表中的物理位置：
// 测试故意让建表列顺序 (b, a, y, x) 与主键 (a, b)、唯一键 (x, y) 声明顺序相反，
// 若某方言内省用错 ordinal 来源（如 PG 的 key_column_usage.ordinal_position 是表内位置），断言会失败。
func assertKeyColumnOrder(t *testing.T, keys []dbi.KeyConstraint, kt dbi.KeyType, wantCols []string) {
	t.Helper()
	for _, k := range keys {
		if k.Type != kt || len(k.Columns) != len(wantCols) {
			continue
		}
		ok := true
		for i, c := range k.Columns {
			if c.Name != wantCols[i] || c.Ordinal != i+1 {
				ok = false
				break
			}
		}
		if ok {
			return
		}
	}
	t.Fatalf("未找到 %s 键且列序为 %v（含 ordinal 1..n），实际键集=%+v", kt, wantCols, keys)
}

func TestITMysqlTableKeys(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()
	table := "it_t_keys"
	q := conn.GetDialect().Quoter().QuoteIdent
	defer func() { _, _ = conn.Exec("DROP TABLE IF EXISTS " + q(table)) }()
	mustExec(t, conn, "DROP TABLE IF EXISTS "+q(table))
	mustExec(t, conn, "CREATE TABLE "+q(table)+
		" (`b` int, `a` int, `y` int, `x` int, PRIMARY KEY (`a`,`b`), UNIQUE KEY `uk_xy` (`x`,`y`))")

	keys, err := conn.Metadata().GetKeys(context.Background(), "", table)
	require.NoError(t, err)

	// 主键存在且列序为定义序 (a, b)，非表内位置 (b, a)
	var pk *dbi.KeyConstraint
	for i := range keys {
		if keys[i].Type == dbi.KeyTypePrimary {
			pk = &keys[i]
		}
	}
	require.NotNil(t, pk, "应存在主键，实际=%+v", keys)
	require.Len(t, pk.Columns, 2)
	assert.Equal(t, "a", pk.Columns[0].Name)
	assert.Equal(t, "b", pk.Columns[1].Name)
	assert.Equal(t, 1, pk.Columns[0].Ordinal)
	assert.Equal(t, 2, pk.Columns[1].Ordinal)

	// 唯一键 (x, y) 亦按定义序返回
	assertKeyColumnOrder(t, keys, dbi.KeyTypeUnique, []string{"x", "y"})
}

func TestITPgTableKeys(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()
	table := "it_t_keys"
	q := conn.GetDialect().Quoter().QuoteIdent
	defer func() { _, _ = conn.Exec("DROP TABLE IF EXISTS " + q(table)) }()
	mustExec(t, conn, "DROP TABLE IF EXISTS "+q(table))
	mustExec(t, conn, "CREATE TABLE "+q(table)+
		` ("b" int, "a" int, "y" int, "x" int, PRIMARY KEY ("a","b"), UNIQUE ("x","y"))`)

	keys, err := conn.Metadata().GetKeys(context.Background(), "", table)
	require.NoError(t, err)

	var pk *dbi.KeyConstraint
	for i := range keys {
		if keys[i].Type == dbi.KeyTypePrimary {
			pk = &keys[i]
		}
	}
	require.NotNil(t, pk, "应存在主键，实际=%+v", keys)
	require.Len(t, pk.Columns, 2)
	assert.Equal(t, "a", pk.Columns[0].Name)
	assert.Equal(t, "b", pk.Columns[1].Name)
	assert.Equal(t, 1, pk.Columns[0].Ordinal)
	assert.Equal(t, 2, pk.Columns[1].Ordinal)

	// PG 若误用 key_column_usage.ordinal_position（表内位置）会得到 (b, a)；此处必须为定义序 (x, y)
	assertKeyColumnOrder(t, keys, dbi.KeyTypeUnique, []string{"x", "y"})
}

// 无主键表：GetKeys 应返回不含主键的结果（不报错、不虚构）。
func TestITMysqlTableKeysNoPk(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()
	table := "it_t_nopk"
	q := conn.GetDialect().Quoter().QuoteIdent
	defer func() { _, _ = conn.Exec("DROP TABLE IF EXISTS " + q(table)) }()
	mustExec(t, conn, "DROP TABLE IF EXISTS "+q(table))
	mustExec(t, conn, "CREATE TABLE "+q(table)+" (`id` int, `name` varchar(20))")

	keys, err := conn.Metadata().GetKeys(context.Background(), "", table)
	require.NoError(t, err)
	for _, k := range keys {
		assert.NotEqual(t, dbi.KeyTypePrimary, k.Type, "无主键表不应返回主键约束")
	}
}

func TestITSQLiteTableKeys(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()
	table := "it_t_keys"
	q := conn.GetDialect().Quoter().QuoteIdent
	defer func() { _, _ = conn.Exec("DROP TABLE IF EXISTS " + q(table)) }()
	mustExec(t, conn, "DROP TABLE IF EXISTS "+q(table))
	// 列声明顺序 (b,a,y,x) 与主键 (a,b)、唯一键 (x,y) 相反，验证键内序而非物理序
	mustExec(t, conn, "CREATE TABLE "+q(table)+
		" (b INTEGER, a INTEGER, y INTEGER, x INTEGER, PRIMARY KEY (a, b), UNIQUE (x, y))")

	keys, err := conn.Metadata().GetKeys(context.Background(), "", table)
	require.NoError(t, err)

	var pk *dbi.KeyConstraint
	for i := range keys {
		if keys[i].Type == dbi.KeyTypePrimary {
			pk = &keys[i]
		}
	}
	require.NotNil(t, pk, "应存在主键，实际=%+v", keys)
	require.Len(t, pk.Columns, 2)
	assert.Equal(t, "a", pk.Columns[0].Name)
	assert.Equal(t, "b", pk.Columns[1].Name)

	// 唯一键 (x, y) 有序；主键自动索引(origin='pk')不应被当作唯一键
	assertKeyColumnOrder(t, keys, dbi.KeyTypeUnique, []string{"x", "y"})
}
