package itest

// 生成列（MySQL VIRTUAL/STORED GENERATED、pg STORED生成列与IDENTITY标识列、SQL Server计算列）
// 跨方言真实往返集成测试。
//
// 该链路有两个必须同时成立的性质（任一侧判错都会造成静默数据失真）：
//  1. 同方言：目标表必须仍被建成生成列，且INSERT列集必须剔除该列（否则硬报错或值退化为NULL）；
//  2. 异构：目标表无法重建派生表达式，该列必须是普通列且必须插入源库派生出的值（否则值静默变NULL）。
//
// 另覆盖：冲突处理（upsert/merge）子句不得包含生成列/标识列；SQL Server的MERGE以分号结尾。
//
// 运行方式：cd server && go test -tags it -count=1 -run TestITGeneratedColumn ./internal/db/dbm/

import (
	"mayfly-go/internal/db/dbm"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

const (
	gcMysql    = dbi.DbType("mysql")
	gcPostgres = dbi.DbType("postgres")
	gcMssql    = dbi.DbType("mssql")
)

const itMssqlDatabase = "mayfly_it"

// gcMssqlConn 连接本机SQL Server集成测试容器；容器未启动时跳过用例（不影响其余方言回归）
func gcMssqlConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	conn, err := dbm.Conn(itCtx(), &dbi.DbInfo{
		Type: gcMssql, Host: "127.0.0.1", Port: 11433,
		Username: "sa", Password: "Mayfly_123456",
		Database: itMssqlDatabase + "/dbo", Params: "encrypt=disable",
	})
	if err != nil {
		t.Skipf("mssql it container unavailable: %s", err.Error())
	}
	if err = conn.Ping(); err != nil {
		_ = conn.Close()
		t.Skipf("mssql it container unreachable: %s", err.Error())
	}
	return conn
}

func gcConn(t *testing.T, dt dbi.DbType) *dbi.DbConn {
	t.Helper()
	switch dt {
	case gcMysql:
		return mysqlConn(t)
	case gcPostgres:
		return pgConn(t)
	default:
		return gcMssqlConn(t)
	}
}

func gcShort(dt dbi.DbType) string {
	switch dt {
	case gcMysql:
		return "my"
	case gcPostgres:
		return "pg"
	default:
		return "ms"
	}
}

// gcSourceDDL 各方言手写生成列源表：两个派生列（整型求和、含常量/类型转换的表达式），
// 表达式刻意使用带括号与函数的形态，验证元数据取到的是原文而非被截断/改写的内容
func gcSourceDDL(dt dbi.DbType, table string) string {
	switch dt {
	case gcMysql:
		return fmt.Sprintf("CREATE TABLE `%s` (\n"+
			"  id int NOT NULL,\n"+
			"  a int NOT NULL,\n"+
			"  b int NOT NULL,\n"+
			"  g_sum int GENERATED ALWAYS AS ((`a` + `b`)) STORED,\n"+
			"  g_cat varchar(100) GENERATED ALWAYS AS (concat(`a`, '-', `b`)) VIRTUAL,\n"+
			"  PRIMARY KEY (id)\n"+
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", table)
	case gcPostgres:
		return fmt.Sprintf(`CREATE TABLE "%s" (
  id int NOT NULL PRIMARY KEY,
  a int NOT NULL,
  b int NOT NULL,
  g_sum int GENERATED ALWAYS AS (a + b) STORED,
  g_rate numeric(18,2) GENERATED ALWAYS AS ((b::numeric * 1.5)) STORED
)`, table)
	default:
		return fmt.Sprintf("CREATE TABLE [%s] (\n"+
			"  [id] int NOT NULL PRIMARY KEY,\n"+
			"  [a] int NOT NULL,\n"+
			"  [b] int NOT NULL,\n"+
			"  [g_sum] AS ([a] + [b]),\n"+
			"  [g_rate] AS (CAST([b] AS numeric(18,2)) * 1.5) PERSISTED\n"+
			")", table)
	}
}

// gcBaseInsert 源表基础数据（只写物理列，生成列由库自身派生）
func gcBaseInsert(dt dbi.DbType, table string) string {
	quote := func(name string) string {
		switch dt {
		case gcMysql:
			return "`" + name + "`"
		case gcPostgres:
			return "\"" + name + "\""
		default:
			return "[" + name + "]"
		}
	}
	cols := fmt.Sprintf("%s, %s, %s", quote("id"), quote("a"), quote("b"))
	tab := quote(table)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (1, 10, 20), (2, -5, 7)", tab, cols)
}

// gcExpect 源/目标行期望值：id -> (g_sum, g_rate/g_cat)
func gcExpect(dt dbi.DbType, id int64) (sum, rate string) {
	var a, b int64
	switch id {
	case 1:
		a, b = 10, 20
	default:
		a, b = -5, 7
	}
	if dt == gcMysql {
		return fmt.Sprintf("%d", a+b), fmt.Sprintf("%d-%d", a, b)
	}
	return fmt.Sprintf("%d", a+b), fmt.Sprintf("%.2f", float64(b)*1.5)
}

// gcGeneratedCol 断言元数据中指定列已被识别为生成列，且取到了派生表达式原文
func gcGeneratedCol(t *testing.T, cols []dbi.Column, name string) dbi.Column {
	t.Helper()
	for _, col := range cols {
		if !strings.EqualFold(col.ColumnName, name) {
			continue
		}
		require.True(t, col.IsGenerated, "列 [%s] 未被识别为生成列", name)
		expr, _ := col.Extra[dbi.ColumnExtraGenExpr].(string)
		require.NotEmpty(t, expr, "列 [%s] 的派生表达式未从元数据取到", name)
		genType, _ := col.Extra[dbi.ColumnExtraGenDbType].(string)
		require.NotEmpty(t, genType, "列 [%s] 的表达式源方言未记录", name)
		return col
	}
	t.Fatalf("元数据缺少列 [%s]", name)
	return dbi.Column{}
}

// gcRows 读取表内全部行（按id排序）
func gcRows(t *testing.T, conn *dbi.DbConn, table string) []map[string]any {
	t.Helper()
	_, rows, err := conn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY %s",
		conn.GetDialect().Quoter().QuoteIdent(table), conn.GetDialect().Quoter().QuoteIdent("id")))
	require.NoError(t, err)
	return rows
}

// gcId 取行的id值：驱动可能以[]byte回传数值列（未开启列值转换），必须先归一再取数值
func gcId(row map[string]any) int64 {
	id, _ := toFloat(normalizeDbValue(row["id"]))
	return int64(id)
}

// TestITGeneratedColumnMetadata 各方言生成列元数据捕获（表达式原文、STORED/VIRTUAL、标识列自增）
func TestITGeneratedColumnMetadata(t *testing.T) {
	srcTypes := []dbi.DbType{gcMysql, gcPostgres, gcMssql}
	for _, st := range srcTypes {
		t.Run(gcShort(st), func(t *testing.T) {
			conn := gcConn(t, st)
			defer conn.Close()
			table := "it_gc_meta_" + gcShort(st)
			defer func() {
				_, _ = conn.Exec("DROP TABLE IF EXISTS " + conn.GetDialect().Quoter().QuoteIdent(table))
			}()

			mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().QuoteIdent(table))
			mustExec(t, conn, gcSourceDDL(st, table))
			mustExec(t, conn, gcBaseInsert(st, table))

			cols, err := conn.GetMetadata().GetColumns(table)
			require.NoError(t, err)
			require.Len(t, cols, 5)

			gotRateCol := "g_rate"
			if st == gcMysql {
				gotRateCol = "g_cat"
			}
			sumCol := gcGeneratedCol(t, cols, "g_sum")
			rateCol := gcGeneratedCol(t, cols, gotRateCol)

			// 物理列不得被误判为生成列
			for _, col := range cols {
				if strings.EqualFold(col.ColumnName, "id") || strings.EqualFold(col.ColumnName, "a") || strings.EqualFold(col.ColumnName, "b") {
					assert.False(t, col.IsGenerated, "普通列 [%s] 被误判为生成列", col.ColumnName)
				}
			}

			// 存储/虚拟形态必须区分：MySQL两个列一STORED一VIRTUAL；pg均为STORED；SQL Server仅PERSISTED算存储
			sumStored, rateStored := dbi.GeneratedColumnStored(sumCol), dbi.GeneratedColumnStored(rateCol)
			switch st {
			case gcMysql:
				assert.True(t, sumStored, "MySQL STORED生成列必须标记为存储")
				assert.False(t, rateStored, "MySQL VIRTUAL生成列不得标记为存储")
			case gcPostgres:
				assert.True(t, sumStored && rateStored, "pg生成列均为STORED")
			default:
				assert.False(t, sumStored, "SQL Server非PERSISTED计算列不得标记为存储")
				assert.True(t, rateStored, "SQL Server PERSISTED计算列必须标记为存储")
			}

			// 派生值本身必须由库算出（防止元数据断言通过而数据错误）
			rows := gcRows(t, conn, table)
			require.Len(t, rows, 2)
			for _, row := range rows {
				wantSum, wantRate := gcExpect(st, gcId(row))
				assertCell(t, "g_sum", wantSum, row["g_sum"])
				assertCell(t, gotRateCol, wantRate, row[gotRateCol])
			}
		})
	}
}

// TestITGeneratedColumnPgIdentity 标识列（GENERATED ALWAYS AS IDENTITY）必须被识别为自增列，
// 否则结构迁移后目标列退化为普通整型列，后续插入不再自动取值
func TestITGeneratedColumnPgIdentity(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()
	table := "it_gc_identity"
	defer func() { _, _ = conn.Exec("DROP TABLE IF EXISTS " + conn.GetDialect().Quoter().QuoteIdent(table)) }()

	mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().QuoteIdent(table))
	mustExec(t, conn, fmt.Sprintf(`CREATE TABLE "%s" (
  id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  seq int GENERATED BY DEFAULT AS IDENTITY,
  v varchar(20)
)`, table))

	cols, err := conn.GetMetadata().GetColumns(table)
	require.NoError(t, err)
	colById := make(map[string]dbi.Column, len(cols))
	for _, col := range cols {
		colById[col.ColumnName] = col
	}
	assert.True(t, colById["id"].AutoIncrement, "主键标识列必须标记为自增列")
	// 非主键标识列不标记自增：迁入MySQL会因「自增列必须是键」直接建表失败（见markGeneratedColumns注释）
	assert.False(t, colById["seq"].AutoIncrement, "非主键标识列不得标记自增")
	assert.False(t, colById["v"].IsGenerated)
}

// TestITGeneratedColumnSameDialect 同方言结构迁移+数据迁移完整往返：目标列必须仍是生成列，
// 且INSERT必须剔除该列（列集与行值同步剔除）
func TestITGeneratedColumnSameDialect(t *testing.T) {
	for _, st := range []dbi.DbType{gcMysql, gcPostgres, gcMssql} {
		st := st
		t.Run(gcShort(st)+"2"+gcShort(st), func(t *testing.T) {
			conn := gcConn(t, st)
			defer conn.Close()
			srcTable := "it_gc_same_" + gcShort(st) + "_src"
			dstTable := srcTable + "_dst"
			defer func() {
				_, _ = conn.Exec("DROP TABLE IF EXISTS " + conn.GetDialect().Quoter().QuoteIdent(srcTable))
				_, _ = conn.Exec("DROP TABLE IF EXISTS " + conn.GetDialect().Quoter().QuoteIdent(dstTable))
			}()

			mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().QuoteIdent(srcTable))
			mustExec(t, conn, gcSourceDDL(st, srcTable))
			mustExec(t, conn, gcBaseInsert(st, srcTable))

			srcCols, err := conn.GetMetadata().GetColumns(srcTable)
			require.NoError(t, err)

			// 列元数据复制到目标表名（同方言无需类型转换，与迁移链路的srcDbType==targetDbType分支一致）
			cols := make([]dbi.Column, 0, len(srcCols))
			for _, col := range srcCols {
				col.TableName = dstTable
				cols = append(cols, col)
			}

			dialect := conn.GetDialect()
			ddls := dialect.GetSQLGenerator().GenTableDDL(dbi.Table{TableName: dstTable}, cols, true)
			for _, ddl := range ddls {
				mustExec(t, conn, ddl)
			}

			// 结构保真：目标库回读，两个派生列必须仍是生成列（不能退化为普通列）
			dstCols, err := conn.GetMetadata().GetColumns(dstTable)
			require.NoError(t, err)
			rateName := "g_rate"
			if st == gcMysql {
				rateName = "g_cat"
			}
			gcGeneratedCol(t, dstCols, "g_sum")
			gcGeneratedCol(t, dstCols, rateName)

			// 数据保真：用源表行值走GenInsert写入目标表，生成列必须由目标库重新派生
			srcRows := gcRows(t, conn, srcTable)
			values := make([][]any, 0, len(srcRows))
			for _, row := range srcRows {
				vals := make([]any, 0, len(cols))
				for _, col := range cols {
					vals = append(vals, row[strings.ToLower(col.ColumnName)])
				}
				values = append(values, vals)
			}
			for _, sql := range dialect.GetSQLGenerator().GenInsert(dstTable, cols, values, dbi.DuplicateStrategyNone, nil) {
				mustExec(t, conn, sql)
			}

			dstRows := gcRows(t, conn, dstTable)
			require.Len(t, dstRows, 2)
			for _, row := range dstRows {
				wantSum, wantRate := gcExpect(st, gcId(row))
				assertCell(t, "dst.g_sum", wantSum, row["g_sum"])
				assertCell(t, "dst."+rateName, wantRate, row[rateName])
			}
		})
	}
}

// TestITGeneratedColumnHeterogeneous 异构迁移：目标库无法复用源方言表达式，生成列必须建成普通列，
// 且源库派生出的值必须完整落入目标列（不得因剔除而静默变NULL）
func TestITGeneratedColumnHeterogeneous(t *testing.T) {
	pairs := [][2]dbi.DbType{
		{gcMysql, gcPostgres}, {gcPostgres, gcMysql},
		{gcMssql, gcMysql}, {gcMssql, gcPostgres},
	}
	for _, pair := range pairs {
		srcType, dstType := pair[0], pair[1]
		name := gcShort(srcType) + "2" + gcShort(dstType)
		t.Run(name, func(t *testing.T) {
			srcConn := gcConn(t, srcType)
			dstConn := srcConn
			if dstType != srcType {
				dstConn = gcConn(t, dstType)
			}
			require.NotNil(t, srcConn)
			require.NotNil(t, dstConn)
			t.Cleanup(func() { _ = srcConn.Close() })
			if dstConn != srcConn {
				t.Cleanup(func() { _ = dstConn.Close() })
			}

			srcTable := "it_gc_het_" + gcShort(srcType) + gcShort(dstType) + "_src"
			dstTable := srcTable + "_dst"
			t.Cleanup(func() {
				_, _ = srcConn.Exec("DROP TABLE IF EXISTS " + srcConn.GetDialect().Quoter().QuoteIdent(srcTable))
				_, _ = dstConn.Exec("DROP TABLE IF EXISTS " + dstConn.GetDialect().Quoter().QuoteIdent(dstTable))
			})

			mustExec(t, srcConn, "DROP TABLE IF EXISTS "+srcConn.GetDialect().Quoter().QuoteIdent(srcTable))
			mustExec(t, srcConn, gcSourceDDL(srcType, srcTable))
			mustExec(t, srcConn, gcBaseInsert(srcType, srcTable))

			srcCols, err := srcConn.GetMetadata().GetColumns(srcTable)
			require.NoError(t, err)

			dstDialect := dstConn.GetDialect()
			cols := make([]dbi.Column, 0, len(srcCols))
			for _, col := range srcCols {
				col.TableName = dstTable
				require.NoError(t, dbi.ConvToTargetDbColumn(srcConn.Info.Type, dstConn.Info.Type, dstDialect, &col),
					"convert column [%s] failed", col.ColumnName)
				cols = append(cols, col)
			}

			for _, ddl := range dstDialect.GetSQLGenerator().GenTableDDL(dbi.Table{TableName: dstTable}, cols, true) {
				mustExec(t, dstConn, ddl)
			}

			// 目标列必须是普通列（异构不重建派生表达式），且无NULL默认值隐患
			dstCols, err := dstConn.GetMetadata().GetColumns(dstTable)
			require.NoError(t, err)
			require.Len(t, dstCols, len(cols), "异构迁移后目标列数不得变化（生成列不得被丢弃）")
			for _, col := range dstCols {
				assert.False(t, col.IsGenerated, "异构目标列 [%s] 不应为生成列", col.ColumnName)
			}

			srcRows := gcRows(t, srcConn, srcTable)
			values := make([][]any, 0, len(srcRows))
			for _, row := range srcRows {
				vals := make([]any, 0, len(cols))
				for _, col := range cols {
					vals = append(vals, row[strings.ToLower(col.ColumnName)])
				}
				values = append(values, vals)
			}
			for _, sql := range dstDialect.GetSQLGenerator().GenInsert(dstTable, cols, values, dbi.DuplicateStrategyNone, nil) {
				mustExec(t, dstConn, sql)
			}

			rateName := "g_rate"
			if srcType == gcMysql {
				rateName = "g_cat"
			}
			dstRows := gcRows(t, dstConn, dstTable)
			require.Len(t, dstRows, 2)
			for _, row := range dstRows {
				wantSum, wantRate := gcExpect(srcType, gcId(row))
				assertCell(t, name+".dst.g_sum", wantSum, row["g_sum"])
				assertCell(t, name+".dst."+rateName, wantRate, row[rateName])
			}
		})
	}
}

// TestITGeneratedColumnUpsert 冲突处理（更新策略）：目标表元数据含生成列时，
// upsert/merge 的SET与INSERT子句都不得包含该列，否则整条语句报错使同步全量失败
func TestITGeneratedColumnUpsert(t *testing.T) {
	for _, st := range []dbi.DbType{gcMysql, gcPostgres, gcMssql} {
		st := st
		t.Run(gcShort(st), func(t *testing.T) {
			conn := gcConn(t, st)
			defer conn.Close()
			table := "it_gc_upsert_" + gcShort(st)
			defer func() {
				_, _ = conn.Exec("DROP TABLE IF EXISTS " + conn.GetDialect().Quoter().QuoteIdent(table))
			}()

			mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().QuoteIdent(table))
			mustExec(t, conn, gcSourceDDL(st, table))
			mustExec(t, conn, gcBaseInsert(st, table))

			// 与数据同步链路一致：列集与目标表元信息全部取自目标表自身元数据
			cols, err := conn.GetMetadata().GetColumns(table)
			require.NoError(t, err)
			rateName := "g_rate"
			if st == gcMysql {
				rateName = "g_cat"
			}

			// 新值：id=1的a/b改为100/200，id=3为新行；生成列传入陈旧值，若被写入SET则结果不会变化
			newBase := map[int64][2]int64{1: {100, 200}, 3: {5, 6}}
			values := make([][]any, 0, len(newBase))
			ids := make([]int64, 0, len(newBase))
			for _, id := range []int64{1, 3} {
				ids = append(ids, id)
				base := newBase[id]
				vals := make([]any, 0, len(cols))
				for _, col := range cols {
					switch strings.ToLower(col.ColumnName) {
					case "id":
						vals = append(vals, id)
					case "a":
						vals = append(vals, base[0])
					case "b":
						vals = append(vals, base[1])
					case "g_sum":
						vals = append(vals, -1)
					default:
						vals = append(vals, "stale")
					}
				}
				values = append(values, vals)
			}

			meta := &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}
			sqls := conn.GetDialect().GetSQLGenerator().GenInsert(table, cols, values, dbi.DuplicateStrategyUpdate, meta)
			require.NotEmpty(t, sqls)
			for _, sql := range sqls {
				assert.NotContains(t, strings.ToUpper(strings.Join(strings.Fields(sql), " ")), strings.ToUpper("= excluded."+rateName),
					"upsert的SET子句不得更新生成列 [%s]: %s", rateName, sql)
				mustExec(t, conn, sql)
			}

			rows := gcRows(t, conn, table)
			require.Len(t, rows, 3, "id=1应被更新、id=3应被插入")
			byId := make(map[int64]map[string]any, len(rows))
			for _, row := range rows {
				byId[gcId(row)] = row
			}
			assertCell(t, "a", int64(100), byId[1]["a"])
			assertCell(t, "b", int64(200), byId[1]["b"])
			// 派生列由目标库重算：a+b=300，说明陈旧值未被写入且表达式仍生效
			assertCell(t, "g_sum", int64(300), byId[1]["g_sum"])
			assertCell(t, "g_sum", int64(11), byId[3]["g_sum"])
			if st == gcMysql {
				assertCell(t, "g_cat", "100-200", byId[1]["g_cat"])
			} else {
				assertCell(t, "g_rate", "300.00", byId[1]["g_rate"])
				assertCell(t, "g_rate", "9.00", byId[3]["g_rate"])
			}
		})
	}
}

// TestITGeneratedColumnLongExpr 超长派生表达式（超过information_schema.COLUMNS.GENERATION_EXPRESSION
// 的4096字符截断阈值）：元数据链路必须取SHOW CREATE TABLE的完整原文（不受information_schema截断影响），
// 同方言迁移时应完整重建生成列而非退化，且目标库重算值与源值一致
func TestITGeneratedColumnLongExpr(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()
	table := "it_gc_longexpr"
	defer func() { _, _ = conn.Exec("DROP TABLE IF EXISTS " + conn.GetDialect().Quoter().QuoteIdent(table)) }()

	// concat 15个定宽片段：表达式文本约4198字符>4096（information_schema截断阈值）；
	// 派生值15*276=4140字符，列长需容下其值（值超列长会使源表INSERT本身报1406），故列长取6000
	parts := make([]string, 0, 15)
	for i := 0; i < 15; i++ {
		parts = append(parts, fmt.Sprintf("'%0276d'", i))
	}
	exprText := fmt.Sprintf("concat(%s)", strings.Join(parts, ", "))
	mustExec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().QuoteIdent(table))
	mustExec(t, conn, fmt.Sprintf("CREATE TABLE `%s` (\n  id int NOT NULL,\n  a int NOT NULL,\n  g_long varchar(6000) GENERATED ALWAYS AS (%s) STORED,\n  PRIMARY KEY (id)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		table, exprText))
	mustExec(t, conn, fmt.Sprintf("INSERT INTO `%s` (id, a) VALUES (1, 1)", table))

	cols, err := conn.GetMetadata().GetColumns(table)
	require.NoError(t, err)
	var longCol dbi.Column
	found := false
	for _, col := range cols {
		if col.ColumnName == "g_long" {
			longCol, found = col, true
		}
	}
	require.True(t, found)
	assert.True(t, longCol.IsGenerated, "生成列必须被识别")
	// SHOW CREATE TABLE原文完整可达：超长表达式也必须可重建（同方言）
	assert.True(t, dbi.PreservableGeneratedColumn(longCol, gcMysql),
		"超长表达式经SHOW CREATE TABLE原文取得，必须仍可同方言重建")
	// MySQL的SHOW CREATE TABLE会规范化输出：字面量前缀变为_utf8mb4、逗号后空格被压缩，
	// 故归一化后比对（去除字符集前缀与空白）验证15个片段全部在位、原文未截断
	gotExpr := dbi.GeneratedColumnExpr(longCol)
	normExpr := func(s string) string {
		s = strings.ReplaceAll(s, "_utf8mb4", "")
		return strings.ReplaceAll(s, " ", "")
	}
	require.Greater(t, len(gotExpr), 4096, "超过information_schema截断阈值的表达式必须完整取得")
	assert.Equal(t, normExpr(exprText), normExpr(gotExpr), "派生表达式归一化后必须完整无截断")

	// 同方言重建路径：目标表把g_long原样建成生成列，插入时剔除该列，由目标库重算
	dstTable := table + "_dst"
	defer func() { _, _ = conn.Exec("DROP TABLE IF EXISTS " + conn.GetDialect().Quoter().QuoteIdent(dstTable)) }()
	dstCols := make([]dbi.Column, 0, len(cols))
	for _, col := range cols {
		col.TableName = dstTable
		dstCols = append(dstCols, col)
	}
	for _, ddl := range conn.GetDialect().GetSQLGenerator().GenTableDDL(dbi.Table{TableName: dstTable}, dstCols, true) {
		mustExec(t, conn, ddl)
	}
	_, srcRows, err := conn.Query("SELECT * FROM `" + table + "`")
	require.NoError(t, err)
	values := make([][]any, 0, len(srcRows))
	for _, row := range srcRows {
		vals := make([]any, 0, len(dstCols))
		for _, col := range dstCols {
			vals = append(vals, row[strings.ToLower(col.ColumnName)])
		}
		values = append(values, vals)
	}
	for _, sql := range conn.GetDialect().GetSQLGenerator().GenInsert(dstTable, dstCols, values, dbi.DuplicateStrategyNone, nil) {
		mustExec(t, conn, sql)
	}
	_, dstRows, err := conn.Query("SELECT * FROM `" + dstTable + "`")
	require.NoError(t, err)
	require.Len(t, dstRows, 1)
	require.NotNil(t, dstRows[0]["g_long"], "生成列重建后目标库必须重算出值，不得静默变NULL")
	assert.Equal(t, 15*276, len(fmt.Sprintf("%v", dstRows[0]["g_long"])), "目标库重算值应与源派生值完全一致")
}
