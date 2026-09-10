package itest

// 数据同步链路的复杂值与特殊标识符真实集成测试。
//
// 与dump/import（导出脚本→切割→导入）不同，数据同步走 srcData2TargetDb：
// 源库查询 → 字段映射 → 目标方言GenInsert（含冲突策略与TargetTableMeta唯一列）→ 目标库事务执行。
// 冲突列（主键/唯一列）与插入列都要按目标方言引用，复杂字符串值要按目标方言字面量转义，
// 任一环节处理不当会使含空格/分号/引号的列名或值同步失败、错列或失真。
//
// 覆盖策略：全量插入（None）、冲突覆盖（Update/upsert）、冲突忽略（Ignore），
// 并验证冲突后新增行仍能同步。
//
// 运行：cd server && go test -tags it -count=1 -run TestITDataSyncComplex ./internal/db/application/

import (
	"mayfly-go/internal/db/application/transfer"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbsync "mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

// itRunDataSync 以指定冲突策略把源表数据同步到目标表（复用产品同步实现）
func itRunDataSync(t *testing.T, srcConn, tgtConn *dbi.DbConn, table, pk string, cols []string, strategy int) {
	t.Helper()
	srcQuote := srcConn.GetDialect().Quoter().QuoteIdent
	_, srcRes, err := srcConn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY %s", srcQuote(table), srcQuote(pk)))
	require.NoError(t, err, "源表查询失败")

	allCols := append([]string{pk}, cols...)
	fieldMap := make([]map[string]string, 0, len(allCols))
	for _, c := range allCols {
		fieldMap = append(fieldMap, map[string]string{"src": c, "target": c})
	}

	// 目标插入列以目标库真实结构语义构造（主键列用于冲突检测）
	tgtColumns := make([]dbi.Column, 0, len(allCols))
	for _, c := range allCols {
		col := dbi.Column{ColumnName: c, DataType: "text", Nullable: true}
		if c == pk {
			col.DataType = "int"
			col.IsPrimaryKey = true
			col.Nullable = false
		}
		tgtColumns = append(tgtColumns, col)
	}
	meta := dbi.BuildTargetTableMeta(tgtConn, table, tgtColumns)
	require.NotEmpty(t, meta.UniqueColumns, "冲突检测列应为表主键")

	task := &entity.DataSyncTask{TargetTableName: table, DuplicateStrategy: strategy}
	app := &dbsync.DataSyncAppImpl{}
	require.NoError(t, app.SyncBatch(context.Background(), srcRes, fieldMap, "", task, tgtConn, tgtColumns, meta))
}

// TestITDataSyncComplexValuesAndNames 复杂值 + 特殊列名/主键名的数据同步全链路
func TestITDataSyncComplexValuesAndNames(t *testing.T) {
	cases := []struct {
		src, tgt itDialectNode
		strategy int
		wantUpd  bool // 冲突时是否应覆盖目标已有值
	}{
		{itMysql, itPg, dbi.DuplicateStrategyUpdate, true},
		{itPg, itMysql, dbi.DuplicateStrategyUpdate, true},
		{itPg, itSqlite, dbi.DuplicateStrategyUpdate, true},
		{itMysql, itSqlite, dbi.DuplicateStrategyIgnore, false},
		{itMysql, itPg, dbi.DuplicateStrategyIgnore, false},
	}

	cols := make([]string, 0, len(itSpecialColNames))
	for _, c := range itSpecialColNames {
		cols = append(cols, "c_"+c.suffix)
	}
	// 冲突时用于验证覆盖/忽略语义的列与主键值
	conflictCol := "c_单'引号"
	const conflictPk = 3

	for _, c := range cases {
		name := fmt.Sprintf("%s->%s/%s", c.src.name, c.tgt.name, strategyName(c.strategy))
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			srcConn, tgtConn := c.src.conn(t), c.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()
			table := fmt.Sprintf("it_cxsync_%s_%d", c.src.name, c.strategy)

			itCreateSpecialColumnTable(t, srcConn, table, itSpecialPkName, cols)
			wantRows := itInsertSpecialColumnRows(t, srcConn, table, itSpecialPkName, cols)

			// 目标表结构来自真实dump（同表名，含特殊列名与特殊主键名）
			script := itDumpTable(t, srcConn, table, c.tgt.dbType, true, false, "")
			app := &transfer.DbTransferAppImpl{}
			require.NoError(t, app.ImportDumpStream(ctx, 0, tgtConn, strings.NewReader(script)),
				"导入结构失败, dump脚本:\n%s", itTruncate(script, 6000))

			// 首次全量同步
			itRunDataSync(t, srcConn, tgtConn, table, itSpecialPkName, cols, dbi.DuplicateStrategyNone)
			srcRows := itReadRowsByPk(t, srcConn, table, itSpecialPkName)
			tgtRows := itReadRowsByPk(t, tgtConn, table, itSpecialPkName)
			require.Len(t, tgtRows, wantRows, "首次同步行数不一致")
			assertComplexRowsEqual(t, srcRows, tgtRows, cols)

			// 源侧篡改冲突行的复杂值（含单双引号、分号、反斜杠、换行），再按策略同步
			changed := "同步后新值 it's \"; 分号\\反斜杠\n第二行"
			itSetColumnParamByName(t, srcConn, table, itSpecialPkName, conflictCol, conflictPk, changed)
			itRunDataSync(t, srcConn, tgtConn, table, itSpecialPkName, cols, c.strategy)

			tgtRows = itReadRowsByPk(t, tgtConn, table, itSpecialPkName)
			assert.Len(t, tgtRows, wantRows, "冲突同步不应产生重复行")
			got := itTextAt(findRowByPk(tgtRows, itSpecialPkName, conflictPk), strings.ToLower(conflictCol))
			if c.wantUpd {
				assert.Equal(t, changed, got, "覆盖策略下冲突行必须更新为新值")
			} else {
				assert.NotEqual(t, changed, got, "忽略策略下冲突行必须保持目标原值")
			}

			// 新增行必须能继续同步（策略不影响非冲突行插入），参数化写入以保证值就是原始字符串
			newPk := wantRows + 100
			newVal := "新增复杂值 'q';\\x"
			srcQ := srcConn.GetDialect().Quoter().QuoteIdent
			_, err := srcConn.Exec(fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (%s, %s)", srcQ(table), srcQ(itSpecialPkName),
				srcQ(conflictCol), itPlaceholder(srcConn, 1), itPlaceholder(srcConn, 2)), newPk, newVal)
			require.NoError(t, err, "源库新增行失败")
			itRunDataSync(t, srcConn, tgtConn, table, itSpecialPkName, cols, c.strategy)

			tgtRows = itReadRowsByPk(t, tgtConn, table, itSpecialPkName)
			assert.Len(t, tgtRows, wantRows+1, "新增行必须同步到目标库")
			newRow := findRowByPk(tgtRows, itSpecialPkName, newPk)
			require.NotNil(t, newRow, "目标库缺少新增行")
			assert.Equal(t, newVal, itTextAt(newRow, strings.ToLower(conflictCol)), "新增行复杂值失真")

			_, _ = srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcConn.GetDialect().Quoter().QuoteIdent(table)))
			_, _ = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tgtConn.GetDialect().Quoter().QuoteIdent(table)))
		})
	}
}

func strategyName(strategy int) string {
	switch strategy {
	case dbi.DuplicateStrategyUpdate:
		return "update"
	case dbi.DuplicateStrategyIgnore:
		return "ignore"
	default:
		return "none"
	}
}

// itSetColumnParamByName 按主键更新指定列（主键列名也可能是特殊名称）
func itSetColumnParamByName(t *testing.T, conn *dbi.DbConn, table, pk, column string, pkVal int, val any) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s = %s", quote(table), quote(column),
		itPlaceholder(conn, 1), quote(pk), itPlaceholder(conn, 2)), val, pkVal)
	require.NoError(t, err, "更新列[%s]失败", column)
}

func findRowByPk(rows []map[string]any, pk string, pkVal int) map[string]any {
	key := strings.ToLower(pk)
	for _, row := range rows {
		if v, ok := dbi.ValToInt64(row[key]); ok && v == int64(pkVal) {
			return row
		}
	}
	return nil
}

// assertComplexRowsEqual 按主键对齐后逐列比对复杂文本值
func assertComplexRowsEqual(t *testing.T, srcRows, tgtRows []map[string]any, cols []string) {
	t.Helper()
	tgtByPk := make(map[int64]map[string]any, len(tgtRows))
	for _, row := range tgtRows {
		if v, ok := dbi.ValToInt64(row[strings.ToLower(itSpecialPkName)]); ok {
			tgtByPk[v] = row
		}
	}
	for _, srcRow := range srcRows {
		pkVal, ok := dbi.ValToInt64(srcRow[strings.ToLower(itSpecialPkName)])
		require.True(t, ok, "源行缺少主键值")
		tgtRow, exist := tgtByPk[pkVal]
		require.True(t, exist, "目标库缺少主键[%d]的行", pkVal)
		for _, col := range cols {
			assert.Equal(t, itTextAt(srcRow, strings.ToLower(col)), itTextAt(tgtRow, strings.ToLower(col)),
				"主键[%d]列[%s]同步失真", pkVal, col)
		}
	}
}
