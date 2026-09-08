//go:build it

package transfer

// 特殊标识符与DDL复杂字面量的跨方言真实链路集成测试（结构迁移维度）。
//
// 与复杂字符串数据迁移（db_transfer_complexstrings_it_test.go）互补，此处覆盖结构迁移链路：
// 元数据读取（列名/索引名/表注释/列注释/默认值）→ 目标方言GenTableDDL/GenIndexDDL生成
// （标识符引用 + 字面量转义）→ dump脚本注释头 → 目标方言切割器 → 导入执行 →
// 目标库元数据回读注释、并实测默认值是否按原值生效。
//
// 覆盖形态：列名含空格/中文/单双引号/反引号/分号/注释符/反斜杠/通配符，特殊主键列名，
// 特殊索引名；表与列注释及默认值含单双引号、反斜杠、分号、--、换行；表名含换行（防止
// dump脚本的--注释头被名称中的换行截断而注入可执行语句）。
//
// 运行：cd server && go test -tags it -count=1 -run 'TestITSpecialColumnNames|TestITComplexDdlLiterals|TestITDumpScriptHeader' ./internal/db/application/

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// itSpecialColNames 特殊列名后缀样本
var itSpecialColNames = []struct{ suffix, desc string }{
	{"中文 列", "含空格与中文"},
	{"单'引号", "含单引号"},
	{`双"引号`, "含双引号"},
	{"反`引号", "含反引号"},
	{"分;号", "含分号（切割器最易错切）"},
	{"横线--x", "含行注释符"},
	{"块/**/x", "含块注释符"},
	{`反\斜杠`, "含反斜杠"},
	{"通配%_", "含LIKE通配符"},
}

// itSpecialPkName 特殊主键列名（主键列会流经dump的ORDER BY、校验器的抽样与比对SQL）
const itSpecialPkName = "id 主键;号"

// itSpecialIndexName 特殊索引名
const itSpecialIndexName = "idx;名 --x"

// itLit 按方言的字面量引用与转义（mysql反斜杠是转义符，需额外双写；其余为标准SQL单引号双写）
func itLit(conn *dbi.DbConn, val string) string {
	if conn.Info.Type == "mysql" {
		return "'" + dbi.QuoteEscapeBackslash(val) + "'"
	}
	return "'" + dbi.QuoteEscape(val) + "'"
}

// itCreateSpecialColumnTable 建含特殊列名、特殊主键名、特殊索引名的表
func itCreateSpecialColumnTable(t *testing.T, conn *dbi.DbConn, table, pk string, cols []string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", quote(table)))
	switch conn.Info.Type {
	case "mysql":
		b.WriteString(fmt.Sprintf("  %s int NOT NULL,\n", quote(pk)))
		for _, col := range cols {
			b.WriteString(fmt.Sprintf("  %s text NULL,\n", quote(col)))
		}
		b.WriteString(fmt.Sprintf("  PRIMARY KEY (%s),\n  KEY %s (%s)\n", quote(pk), quote(itSpecialIndexName), quote(pk)))
		b.WriteString(") DEFAULT CHARSET=utf8mb4")
	case "postgres":
		b.WriteString(fmt.Sprintf("  %s integer NOT NULL,\n", quote(pk)))
		for _, col := range cols {
			b.WriteString(fmt.Sprintf("  %s text NULL,\n", quote(col)))
		}
		b.WriteString(fmt.Sprintf("  PRIMARY KEY (%s)\n", quote(pk)))
		b.WriteString(")")
	default: // sqlite
		b.WriteString(fmt.Sprintf("  %s INTEGER PRIMARY KEY,\n", quote(pk)))
		for i, col := range cols {
			b.WriteString(fmt.Sprintf("  %s TEXT", quote(col)))
			if i < len(cols)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(")")
	}
	_, err = conn.Exec(b.String())
	require.NoError(t, err, "建特殊列名表失败")

	// pg/sqlite索引必须独立语句（mysql内联KEY）
	if conn.Info.Type != "mysql" {
		_, err = conn.Exec(fmt.Sprintf("CREATE INDEX %s ON %s (%s)", quote(itSpecialIndexName), quote(table), quote(pk)))
		require.NoError(t, err, "创建特殊索引失败")
	}
}

// itInsertSpecialColumnRows 参数化写入复杂值：第i行第j列取itComplexTexts[(i+j)%len]，
// 使每个特殊列都承载到全部复杂形态
func itInsertSpecialColumnRows(t *testing.T, conn *dbi.DbConn, table, pk string, cols []string) int {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	allCols := append([]string{pk}, cols...)
	ph := make([]string, 0, len(allCols))
	for i := range allCols {
		ph = append(ph, itPlaceholder(conn, i+1))
	}
	insertSql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(table), strings.Join(quoteList(quote, allCols), ", "), strings.Join(ph, ", "))

	total := len(itComplexTexts)
	for i := 0; i < total; i++ {
		args := make([]any, 0, len(allCols))
		args = append(args, i+1)
		for j := range cols {
			args = append(args, itComplexTexts[(i+j)%total])
		}
		_, err := conn.Exec(insertSql, args...)
		require.NoError(t, err, "写入第%d行失败", i+1)
	}
	return total
}

func quoteList(quote func(string) string, names []string) []string {
	res := make([]string, 0, len(names))
	for _, name := range names {
		res = append(res, quote(name))
	}
	return res
}

// itReadRowsByPk 按特殊主键名排序回读（列名统一小写化，规避驱动大小写差异）
func itReadRowsByPk(t *testing.T, conn *dbi.DbConn, table, pk string) []map[string]any {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY %s", quote(table), quote(pk)))
	require.NoError(t, err, "回读表[%s]失败", table)
	res := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m := make(map[string]any, len(row))
		for k, v := range row {
			m[strings.ToLower(k)] = v
		}
		res = append(res, m)
	}
	return res
}

// TestITSpecialColumnNamesMigrate 特殊列名/主键名/索引名的结构+数据迁移全链路
func TestITSpecialColumnNamesMigrate(t *testing.T) {
	pairs := []struct{ src, tgt itDialectNode }{
		{itMysql, itPg}, {itPg, itMysql}, {itSqlite, itPg}, {itMysql, itSqlite},
	}

	cols := make([]string, 0, len(itSpecialColNames))
	for _, c := range itSpecialColNames {
		cols = append(cols, "c_"+c.suffix)
	}

	for _, p := range pairs {
		t.Run(p.src.name+"->"+p.tgt.name, func(t *testing.T) {
			srcConn, tgtConn := p.src.conn(t), p.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()
			table := "it_cxcol_" + p.src.name

			itCreateSpecialColumnTable(t, srcConn, table, itSpecialPkName, cols)
			wantRows := itInsertSpecialColumnRows(t, srcConn, table, itSpecialPkName, cols)

			script := itDumpTable(t, srcConn, table, p.tgt.dbType, true, true, "")
			app := &DbTransferAppImpl{}
			require.NoError(t, app.importDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
				"导入失败, dump脚本:\n%s", itTruncate(script, 6000))

			// 列名集合与顺序必须完全保真
			srcMetaCols, err := srcConn.GetMetadata().GetColumns(table)
			require.NoError(t, err)
			tgtMetaCols, err := tgtConn.GetMetadata().GetColumns(table)
			require.NoError(t, err)
			assert.Equal(t, lowerColNames(srcMetaCols), lowerColNames(tgtMetaCols), "迁移后列名失真")

			// 主键必须仍是迁移前的特殊主键列（校验器依赖主键抽样）
			srcPk, err := srcConn.GetMetadata().GetPrimaryKey(table)
			require.NoError(t, err)
			tgtPk, err := tgtConn.GetMetadata().GetPrimaryKey(table)
			require.NoError(t, err)
			assert.Equal(t, strings.ToLower(srcPk), strings.ToLower(tgtPk), "主键列名迁移失真")

			// 逐行逐列内容比对
			srcRows := itReadRowsByPk(t, srcConn, table, itSpecialPkName)
			tgtRows := itReadRowsByPk(t, tgtConn, table, itSpecialPkName)
			require.Len(t, tgtRows, wantRows, "特殊列名表行数不一致")
			for i := range srcRows {
				for _, col := range cols {
					assert.Equal(t, itTextAt(srcRows[i], strings.ToLower(col)), itTextAt(tgtRows[i], strings.ToLower(col)),
						"第%d行列[%s]迁移失真", i+1, col)
				}
			}

			// 特殊索引名需在目标库存在（各方言自动生成的内部索引数量不同，仅断言目标含该索引）
			_, err = srcConn.GetMetadata().GetTableIndex(table)
			require.NoError(t, err)
			tgtIdx, err := tgtConn.GetMetadata().GetTableIndex(table)
			require.NoError(t, err)
			assert.Contains(t, lowerIndexNames(tgtIdx), strings.ToLower(itSpecialIndexName),
				"特殊索引名未迁移, 目标索引: %v", lowerIndexNames(tgtIdx))

			// 产品校验器需支持特殊列名/主键名
			res := app.verifyTable(context.Background(), srcConn, tgtConn, table)
			require.Empty(t, res.Err, "校验器失败: %s", res.Err)
			assert.True(t, res.CountMatch, "校验器count应一致")
			assert.Empty(t, res.MismatchPk, "校验器不应误报: %v", res.MismatchPk)

			quote := tgtConn.GetDialect().Quoter().QuoteIdent
			_, _ = srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcConn.GetDialect().Quoter().QuoteIdent(table)))
			_, _ = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
		})
	}
}

func lowerColNames(columns []dbi.Column) []string {
	res := make([]string, 0, len(columns))
	for _, c := range columns {
		res = append(res, strings.ToLower(c.ColumnName))
	}
	return res
}

func lowerIndexNames(indexs []dbi.Index) []string {
	res := make([]string, 0, len(indexs))
	for _, i := range indexs {
		res = append(res, strings.ToLower(i.IndexName))
	}
	return res
}

// DDL复杂字面量样本：注释与默认值均含单双引号、反斜杠、分号、注释符，注释额外含换行
const (
	cxDdlTableComment  = "表'注释\"与\"反斜杠\\path;分号--符号\n第二行"
	cxDdlColumnComment = "列注释 it's \"q\" C:\\tmp;--x"
	cxDdlDefault       = `it's a "def"; x\y`
	cxDdlPlainDefault  = "plain_v"
)

// itCreateDdlLiteralTable 建含复杂表注释/列注释/默认值的表（注释与默认值均为原始值）
func itCreateDdlLiteralTable(t *testing.T, conn *dbi.DbConn, table string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)

	var ddl string
	switch conn.Info.Type {
	case "mysql":
		ddl = fmt.Sprintf("CREATE TABLE %s (\n"+
			"  id int NOT NULL,\n"+
			"  d_plain varchar(50) DEFAULT %s,\n"+
			"  d_cx varchar(500) DEFAULT %s,\n"+
			"  t_cx text COMMENT %s,\n"+
			"  PRIMARY KEY (id)\n"+
			") DEFAULT CHARSET=utf8mb4 COMMENT=%s",
			quote(table), itLit(conn, cxDdlPlainDefault), itLit(conn, cxDdlDefault),
			itLit(conn, cxDdlColumnComment), itLit(conn, cxDdlTableComment))
	case "postgres":
		ddl = fmt.Sprintf("CREATE TABLE %s (\n"+
			"  id integer NOT NULL,\n"+
			"  d_plain varchar(50) DEFAULT %s,\n"+
			"  d_cx varchar(500) DEFAULT %s,\n"+
			"  t_cx text,\n"+
			"  PRIMARY KEY (id)\n"+
			")", quote(table), itLit(conn, cxDdlPlainDefault), itLit(conn, cxDdlDefault))
	default: // sqlite：无注释能力，仅验证默认值字面量
		ddl = fmt.Sprintf("CREATE TABLE %s (\n"+
			"  id INTEGER PRIMARY KEY,\n"+
			"  d_plain TEXT DEFAULT %s,\n"+
			"  d_cx TEXT DEFAULT %s,\n"+
			"  t_cx TEXT\n"+
			")", quote(table), itLit(conn, cxDdlPlainDefault), itLit(conn, cxDdlDefault))
	}
	_, err = conn.Exec(ddl)
	require.NoError(t, err, "建表失败")

	if conn.Info.Type == "postgres" {
		_, err = conn.Exec(fmt.Sprintf("COMMENT ON TABLE %s IS %s", quote(table), itLit(conn, cxDdlTableComment)))
		require.NoError(t, err, "设置表注释失败")
		_, err = conn.Exec(fmt.Sprintf("COMMENT ON COLUMN %s.%s IS %s", quote(table), quote("t_cx"), itLit(conn, cxDdlColumnComment)))
		require.NoError(t, err, "设置列注释失败")
	}
}

// itReadEffectiveDefaults 不指定默认列插入一行，回读实际生效的默认值（验证默认值字面量全链路保真）
func itReadEffectiveDefaults(t *testing.T, conn *dbi.DbConn, table string) (plain, cx string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (id) VALUES (9001)", quote(table)))
	require.NoError(t, err, "插入仅主键行失败")
	_, rows, err := conn.Query(fmt.Sprintf("SELECT d_plain, d_cx FROM %s WHERE id = 9001", quote(table)))
	require.NoError(t, err, "回读默认值失败")
	require.Len(t, rows, 1, "未插入到默认值行")
	return itTextAt(lowerRow(rows[0]), "d_plain"), itTextAt(lowerRow(rows[0]), "d_cx")
}

func lowerRow(row map[string]any) map[string]any {
	res := make(map[string]any, len(row))
	for k, v := range row {
		res[strings.ToLower(k)] = v
	}
	return res
}

// TestITComplexDdlLiteralsMigrate 复杂注释/默认值经dump→import后必须在目标库按原值保真
func TestITComplexDdlLiteralsMigrate(t *testing.T) {
	pairs := []struct{ src, tgt itDialectNode }{
		{itMysql, itPg}, {itPg, itMysql},
	}

	for _, p := range pairs {
		t.Run(p.src.name+"->"+p.tgt.name, func(t *testing.T) {
			srcConn, tgtConn := p.src.conn(t), p.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()
			table := "it_cxcmt_" + p.src.name

			itCreateDdlLiteralTable(t, srcConn, table)

			// 源库基线：默认值必须是原始值本身（否则本测试的期望值无意义）
			srcPlain, srcCx := itReadEffectiveDefaults(t, srcConn, table)
			require.Equal(t, cxDdlPlainDefault, srcPlain, "源库默认值基线不符")
			require.Equal(t, cxDdlDefault, srcCx, "源库复杂默认值基线不符")

			// 仅导出结构，导入目标库
			script := itDumpTable(t, srcConn, table, p.tgt.dbType, true, false, "")
			app := &DbTransferAppImpl{}
			require.NoError(t, app.importDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
				"导入失败, dump脚本:\n%s", itTruncate(script, 6000))

			// 注释回读保真
			tgtTables, err := tgtConn.GetMetadata().GetTables(table)
			require.NoError(t, err)
			require.Len(t, tgtTables, 1, "目标表不存在")
			assert.Equal(t, cxDdlTableComment, tgtTables[0].TableComment, "表注释迁移失真")

			tgtCols, err := tgtConn.GetMetadata().GetColumns(table)
			require.NoError(t, err)
			var gotColComment string
			for _, c := range tgtCols {
				if strings.EqualFold(c.ColumnName, "t_cx") {
					gotColComment = c.ColumnComment
				}
			}
			assert.Equal(t, cxDdlColumnComment, gotColComment, "列注释迁移失真")

			// 默认值必须在目标库按原值生效（含单引号/双引号/反斜杠/分号）
			tgtPlain, tgtCx := itReadEffectiveDefaults(t, tgtConn, table)
			assert.Equal(t, cxDdlPlainDefault, tgtPlain, "目标库普通默认值失真")
			assert.Equal(t, cxDdlDefault, tgtCx, "目标库复杂默认值失真")

			quote := tgtConn.GetDialect().Quoter().QuoteIdent
			_, _ = srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcConn.GetDialect().Quoter().QuoteIdent(table)))
			_, _ = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
		})
	}
}

// itHdrInjectTableName 表名内嵌换行 + 可执行语句：若dump的 -- 注释头未做换行处理，
// 该语句会成为独立SQL被导入执行（表不存在而报错，或更糟：真实执行）
const itHdrInjectTableName = "it_cxhdr_ok\nDROP TABLE no_such_tbl_cxhdr"

// TestITDumpScriptHeaderCommentSafe dump脚本中的表名注释头必须不可注入可执行语句
func TestITDumpScriptHeaderCommentSafe(t *testing.T) {
	pairs := []struct{ src, tgt itDialectNode }{
		{itMysql, itPg}, {itPg, itSqlite}, {itSqlite, itMysql},
	}

	for _, p := range pairs {
		t.Run(p.src.name+"->"+p.tgt.name, func(t *testing.T) {
			srcConn, tgtConn := p.src.conn(t), p.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()

			itCreateComplexTable(t, srcConn, itHdrInjectTableName)
			wantRows := itInsertComplexRows(t, srcConn, itHdrInjectTableName)

			script := itDumpTable(t, srcConn, itHdrInjectTableName, p.tgt.dbType, true, true, "")
			// 用目标方言真实切割器逐句检查：表名中的换行不得使注释头提前结束而多出一条以注入语句开头的SQL
			// （合法的 DROP/CREATE/INSERT 语句中该文本处于引用标识符内部，不会出现在句首）
			var stmts []string
			require.NoError(t, tgtConn.GetDialect().GetSQLSplitter().SplitSQL(strings.NewReader(script), func(stmt string) error {
				stmts = append(stmts, stmt)
				return nil
			}))
			require.NotEmpty(t, stmts)
			for _, stmt := range stmts {
				assert.False(t, strings.HasPrefix(strings.ToUpper(strings.TrimSpace(stmt)), "DROP TABLE NO_SUCH_TBL_CXHDR"),
					"表名中的换行破坏了--注释头，注入了可执行语句: %s", itTruncate(stmt, 200))
			}

			app := &DbTransferAppImpl{}
			require.NoError(t, app.importDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
				"导入失败, dump脚本:\n%s", itTruncate(script, 6000))

			quote := tgtConn.GetDialect().Quoter().QuoteIdent
			_, rows, err := tgtConn.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", quote(itHdrInjectTableName)))
			require.NoError(t, err, "含换行表名的目标表查询失败")
			require.Len(t, rows, 1)
			cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
			require.True(t, ok, "count应为数值: %#v", rows[0]["cnt"])
			assert.Equal(t, int64(wantRows), cnt, "含换行表名迁移行数不应丢失或重复")

			_, _ = srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcConn.GetDialect().Quoter().QuoteIdent(itHdrInjectTableName)))
			_, _ = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(itHdrInjectTableName)))
		})
	}
}
