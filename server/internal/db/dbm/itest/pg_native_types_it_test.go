package itest

// pg 原生扩展类型（uuid / 数组）的跨方言迁移端到端守卫
//
// 存在理由：这两类列曾因未注册在 pg 类型引擎里，异构迁移在「源类型解析」一步整表报错
// （`data type [uuid] not found in type engine`），带 uuid 主键的 pg 表根本迁不出去；
// 结构比对里两侧都「认不出」，只能标比不了。注册后必须同时成立三件事：
//  1. 目标 DDL 真能执行（uuid→varchar、数组→text 的降级形态合法，不是拼出 uuid(36) 这类非法类型）；
//  2. 数据真能搬运且不失真（uuid 文本、数组字面量原文落进目标列）；
//  3. 目标为 pg 时保持原生类型不退化，且字符串列不会被误判成 uuid。
//
// 运行：cd server && go test -count=1 ./internal/db/dbm/itest/

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

const (
	pgNativeSrcTable  = "it_pg_native_src"
	pgNativeDestTable = "it_pg_native_dest"
)

const (
	pgNativeUuid1 = "11111111-2222-3333-4444-555555555555"
	pgNativeUuid2 = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
)

// setupPgNativeSource 建含 uuid 主键、bigint[] 数组与文本列的 pg 源表并写入两行
func setupPgNativeSource(t *testing.T, conn *dbi.DbConn) []dbi.Column {
	t.Helper()
	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(pgNativeSrcTable))
	mustExec(t, conn, fmt.Sprintf(`CREATE TABLE %s (
		id uuid NOT NULL PRIMARY KEY,
		tags bigint[],
		name text
	)`, quote(pgNativeSrcTable)))
	mustExec(t, conn, fmt.Sprintf(
		"INSERT INTO %s (id, tags, name) VALUES ('%s'::uuid, '{1,2,3}'::bigint[], 'first'), ('%s'::uuid, NULL, 'second')",
		quote(pgNativeSrcTable), pgNativeUuid1, pgNativeUuid2))

	cols, err := conn.Metadata().GetColumns(pgNativeSrcTable)
	require.NoError(t, err)
	require.Len(t, cols, 3)
	return cols
}

func TestITPgNativeTypesToMysql(t *testing.T) {
	pconn := pgConn(t)
	defer pconn.Close()
	mconn := mysqlConn(t)
	defer mconn.Close()

	srcCols := setupPgNativeSource(t, pconn)
	srcQuote := pconn.GetDialect().Quoter().Quote
	destQuote := mconn.GetDialect().Quoter().Quote
	defer func() {
		_, _ = pconn.Exec("DROP TABLE IF EXISTS " + srcQuote(pgNativeSrcTable))
		_, _ = mconn.Exec("DROP TABLE IF EXISTS " + destQuote(pgNativeDestTable))
	}()

	// 源侧形态即库侧真实类型名，注册前后都不应变化（注册只影响归类，不改元数据呈现）
	byName := make(map[string]dbi.Column, len(srcCols))
	for _, col := range srcCols {
		byName[col.ColumnName] = col
	}
	assert.Equal(t, "uuid", byName["id"].DataType)
	assert.Equal(t, "_int8", byName["tags"].DataType, "pg 数组列的元数据类型名是 _ + 元素类型名")

	// 1. 类型转换：注册前这两列会在此处报错，整表迁不过去
	destCols := make([]dbi.Column, 0, len(srcCols))
	for _, col := range srcCols {
		require.NoError(t, dbi.ConvToTargetDbColumn("postgres", "mysql", mconn.GetDialect(), &col),
			"列 [%s] 转换失败", col.ColumnName)
		destCols = append(destCols, col)
	}
	// uuid→varchar：降级为字符串承载，长度由 mysql 生成器补默认值（uuid 文本 36 字符，不会截断）
	assert.Equal(t, "varchar", destCols[0].DataType)
	// 数组→text：数组字面量原文即文本，落 text 无损
	assert.Equal(t, "text", destCols[1].DataType)

	// 2. 目标库真实建表：DDL 必须可执行（uuid(36) 之类的拼接会在此处暴露）
	for _, ddl := range mconn.GetDialect().GetSQLGenerator().GenTableDDL(
		dbi.Table{TableName: pgNativeDestTable}, destCols, true) {
		mustExec(t, mconn, ddl)
	}

	// 3. 数据搬运：按目标列类型渲染值并真实插入（源表名用源方言的引用符）
	var values [][]any
	_, err := pconn.WalkTableRows(itCtx(), srcQuote(pgNativeSrcTable), func(row map[string]any, _ []*dbi.QueryColumn) error {
		rowVals := make([]any, 0, len(destCols))
		for _, col := range destCols {
			rowVals = append(rowVals, row[col.ColumnName])
		}
		values = append(values, rowVals)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, values, 2)
	for _, sql := range mconn.GetDialect().GetSQLGenerator().
		GenInsert(pgNativeDestTable, destCols, values, dbi.DuplicateStrategyNone, nil) {
		mustExec(t, mconn, sql)
	}

	// 4. 回读比对：uuid 文本与数组字面量必须原样到达，NULL 仍是 NULL
	rows := readAllRows(t, mconn, pgNativeDestTable, "name")
	require.Len(t, rows, 2)
	byName2 := make(map[string]map[string]any, len(rows))
	for _, row := range rows {
		byName2[fmt.Sprint(row["name"])] = row
	}
	require.Contains(t, byName2, "first")
	require.Contains(t, byName2, "second")
	assert.Equal(t, pgNativeUuid1, fmt.Sprint(byName2["first"]["id"]), "uuid 值必须原样搬运")
	assert.Equal(t, "{1,2,3}", fmt.Sprint(byName2["first"]["tags"]), "数组字面量必须原样搬运")
	assert.Nil(t, byName2["second"]["tags"], "NULL 数组不得被写成空串")
}

// TestITPgTargetKeepsVarcharForStringSources 目标为 pg 时，只有源类别确为 uuid 的列才落 uuid。
//
// 防的是「按形状猜类型」：mysql 的 varchar(36)（很常被用来存 uuid 文本）迁移到 pg 必须仍是
// varchar，若因长度像 uuid 就转成原生 uuid，后续写入非 uuid 文本会直接报错。
// TCUUID → pg 保持原生 uuid 的正向断言见 dialect/postgres 包的同名规则用例（不依赖真库）。
func TestITPgTargetKeepsVarcharForStringSources(t *testing.T) {
	pconn := pgConn(t)
	defer pconn.Close()

	col := &dbi.Column{ColumnName: "code", DataType: "varchar", CharMaxLength: 36}
	require.NoError(t, dbi.ConvToTargetDbColumn("mysql", "postgres", pconn.GetDialect(), col))
	assert.Equal(t, "varchar", col.DataType, "字符串列转入 pg 不得被误判成 uuid")
	assert.Equal(t, 36, col.CharMaxLength, "varchar 长度信息必须保留")
}
