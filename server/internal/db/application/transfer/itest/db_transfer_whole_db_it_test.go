package itest

// 整库「单流」导出→导入真实链路集成测试（多表 + DDL + 数据 + 索引合并在同一SQL流中）。
//
// 与既有IT的分工：
//   - db_transfer_multitable_big_it_test.go：按表逐张 dump/restore（每张表独立文件、独立事务）
//   - 本文件：一次DumpDbScript产出**整个库**的单一流，由一次ImportDumpStream消费。
//     这是最贴近"库级备份恢复"的形态，也是唯一能让批级提交边界（importStmtBatchSize）
//     跨越表边界、让某张表的dump事务包装（注释头+BEGIN）落在未提交批次中间的场景。
//
// 断言两翼：
//   - 成功路：三表行数/聚合值/复杂文本逐值一致，二级索引恢复
//   - 失败路：流末尾追加坏语句 → 事务内所有内容必须整体回滚，不得残留任何部分批次
//     （mysql/pg→mysql/sqlite→mysql 三种目标为mysql的组合会真实触发隐式提交，最具鉴别力）
//
// 运行：cd server && go test -tags it -count=1 -run TestITWholeDb ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
)

// itWdbTables 整库用例的表集合（名字保证在同一库内其它用例之间排序稳定）
var itWdbTables = []string{"wdb_t_alpha", "wdb_t_beta", "wdb_t_gamma"}

// itWdbSpecialTexts 复杂值样本：引号/反斜杠/换行/分号/注释符/中文/emoji/超长
var itWdbSpecialTexts = []string{
	"it's a \"test\"", `back\slash`, "line1\nline2", "semi;colon;again",
	"-- not a comment", "/* not a block */", "emoji😀中文mixed", "",
	strings.Repeat("q'\\;", 120),
}

// itWdbDDL 各方言建表列定义（id主键 + 复杂文本 + 高精度数值 + 可空列）
//
// sqlite侧声明为VARCHAR(n)而非TEXT：异构整库迁移时，sqlite的TEXT列（元数据无长度）映射到
// mysql即为无长度TEXT，而mysql不允许在未指定前缀长度的情况下为TEXT列建索引
// （Error 1170: BLOB/TEXT column used in key specification without a key length），
// 使“文本列上二级索引”这一常规场景无法完成sqlite→mysql迁移；声明带长度的类型可跨方言可表达
//
// mssql侧用NVARCHAR而非VARCHAR：本组用例的样本值含中文/emoji，而SQL Server库默认排序规则
// （本机SQL_Latin1_General_CP1_CI_AS）下varchar/text列会将非ASCII静默转为'?'，与转义正确性无关而干扰断言
var itWdbDDL = map[string]string{
	"mysql":    "(id INT PRIMARY KEY, c_txt VARCHAR(500), c_num DECIMAL(14,4), c_null VARCHAR(50) NULL)",
	"postgres": "(id INT PRIMARY KEY, c_txt VARCHAR(500), c_num NUMERIC(14,4), c_null VARCHAR(50) NULL)",
	"sqlite":   "(id INTEGER PRIMARY KEY, c_txt VARCHAR(500), c_num NUMERIC, c_null VARCHAR(50))",
	"mssql":    "(id INT PRIMARY KEY, c_txt NVARCHAR(500), c_num DECIMAL(14,4), c_null NVARCHAR(50) NULL)",
}

// itWdbSetup 建表+写入含复杂值的行，返回表名→行数
func itWdbSetup(t *testing.T, conn *dbi.DbConn, rowsPerTable int) map[string]int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	counts := make(map[string]int64, len(itWdbTables))

	for _, table := range itWdbTables {
		_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
		require.NoError(t, err)
		ddl := fmt.Sprintf("CREATE TABLE %s %s", quote(table), itWdbDDL[string(conn.Info.Type)])
		if conn.Info.Type == "mysql" {
			ddl += " DEFAULT CHARSET=utf8mb4"
		}
		_, err = conn.Exec(ddl)
		require.NoError(t, err, "[%s] 建表失败", table)

		for i := 1; i <= rowsPerTable; i++ {
			txt := itWdbSpecialTexts[(i-1)%len(itWdbSpecialTexts)]
			var nullVal any
			if i%3 != 0 {
				nullVal = fmt.Sprintf("n%04d", i)
			}
			// 每第5行附带时间字面量样本（以文本形态存在c_txt尾部，避免跨方言时间展示差异干扰）
			if i%5 == 0 {
				txt += "|2026-02-28 23:59:59.123"
			}
			// 列宽上限500字符（mysql/pg按字符计），超长样本按字符截断以保证三库写入同一值
			if r := []rune(txt); len(r) > 480 {
				txt = string(r[:480])
			}
			_, err := conn.Exec(
				fmt.Sprintf("INSERT INTO %s (id, c_txt, c_num, c_null) VALUES (%s, %s, %s, %s)",
					quote(table), itPlaceholder(conn, 1), itPlaceholder(conn, 2), itPlaceholder(conn, 3), itPlaceholder(conn, 4)),
				int64(i), txt, float64(i)*1.0001, nullVal)
			require.NoError(t, err, "[%s] 第%d行写入失败", table, i)
		}

		// 二级索引：整库dump的索引段必须与数据段同流产出
		_, err = conn.Exec(fmt.Sprintf("CREATE INDEX ix_%s_txt ON %s (c_txt)", strings.TrimPrefix(table, "wdb_t_"), quote(table)))
		require.NoError(t, err, "[%s] 建索引失败", table)
		counts[table] = int64(rowsPerTable)
	}
	return counts
}

// itWdbLenFn 文本长度函数：SQL Server只有LEN（LENGTH为无效对象名），其余方言为LENGTH
func itWdbLenFn(conn *dbi.DbConn) string {
	if conn.Info.Type == "mssql" {
		return "LEN"
	}
	return "LENGTH"
}

// itWdbAgg 表级聚合快照：行数、id边界和、文本总长、非空计数
func itWdbAgg(t *testing.T, conn *dbi.DbConn, table string) string {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf(
		"SELECT COUNT(*) AS c, SUM(id) AS sid, SUM(%s(c_txt)) AS stxt, COUNT(c_null) AS cn FROM %s",
		itWdbLenFn(conn), quote(table)))
	require.NoError(t, err, "[%s] 聚合查询失败", table)
	require.Len(t, rows, 1)
	return fmt.Sprintf("c=%s sid=%s stxt=%s cn=%s",
		itWdbCell(rows[0]["c"]), itWdbCell(rows[0]["sid"]), itWdbCell(rows[0]["stxt"]), itWdbCell(rows[0]["cn"]))
}

// itWdbCell 聚合列取值为数字，统一为字符串便于跨方言快照比对
func itWdbCell(v any) string {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprintf("%v", v)
}

// itWdbAllTexts 读回全表文本列（按id排序），用于逐值比对
func itWdbAllTexts(t *testing.T, conn *dbi.DbConn, table string) []string {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT c_txt FROM %s ORDER BY id", quote(table)))
	require.NoError(t, err, "[%s] 读回文本列失败", table)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, itWdbCell(r["c_txt"]))
	}
	return out
}

// itWdbDumpWholeDb 驱动生产DumpDbScript导出整库为单一SQL流
func itWdbDumpWholeDb(t *testing.T, conn *dbi.DbConn, target dbi.DbType, dumpDDL bool) string {
	return itWdbDumpFlags(t, conn, target, dumpDDL, true)
}

// itWdbDumpFlags 按指定结构/数据开关导出整库：与迁移链路 transferTableDDL2Db/transferTableData2Db
// 使用同一组标志位（结构阶段 DumpDDL:true/DumpData:false，数据阶段反之）
func itWdbDumpFlags(t *testing.T, conn *dbi.DbConn, target dbi.DbType, dumpDDL, dumpData bool) string {
	t.Helper()
	var buf strings.Builder
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName:       conn.Info.Database,
		Tables:       append([]string{}, itWdbTables...),
		DumpDDL:      dumpDDL,
		DumpData:     dumpData,
		TargetDbType: target,
		Writer:       &buf,
	}), "整库导出失败")
	script := buf.String()
	require.NotEmpty(t, script)
	return script
}

// itWdbIndexNames 读回表的全部索引名（小写），用于断言索引确实存在/已迁移
func itWdbIndexNames(t *testing.T, conn *dbi.DbConn, table string) []string {
	t.Helper()
	indexs, err := conn.GetMetadata().GetTableIndex(table)
	require.NoError(t, err, "[%s] 读取索引失败", table)
	return lowerIndexNames(indexs)
}

// itWdbDropAll 删除用例涉及的全部表
func itWdbDropAll(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	for _, table := range itWdbTables {
		_, _ = conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	}
}

// itWdbIndexName 表的二级索引名（与itWdbSetup建索引规则一致）
func itWdbIndexName(table string) string {
	return "ix_" + strings.TrimPrefix(table, "wdb_t_") + "_txt"
}

// itWdbRowsSafe 表不可读时视为0行（失败回滚后DDL可能一并回滚）
func itWdbRowsSafe(t *testing.T, conn *dbi.DbConn, table string) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", quote(table)))
	if err != nil {
		t.Logf("[%s] 表不可读（视为0行）：%s", table, err.Error())
		return 0
	}
	cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
	require.True(t, ok)
	return cnt
}

// TestITWholeDbDumpImportSameDialect 同构整库单流备份恢复：三表内容必须逐值一致
func TestITWholeDbDumpImportSameDialect(t *testing.T) {
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			const rows = 300
			itWdbSetup(t, conn, rows)
			wantAgg := map[string]string{}
			wantTexts := map[string][]string{}
			for _, table := range itWdbTables {
				wantAgg[table] = itWdbAgg(t, conn, table)
				wantTexts[table] = itWdbAllTexts(t, conn, table)
			}

			script := itWdbDumpWholeDb(t, conn, node.dbType, true)
			itWdbDropAll(t, conn)

			app := &transfer.DbTransferAppImpl{}
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)),
				"整库单流导入失败, dump脚本:\n%s", itTruncate(script, 6000))

			for _, table := range itWdbTables {
				assert.Equal(t, wantAgg[table], itWdbAgg(t, conn, table), "[%s] 聚合快照不一致", table)
				assert.Equal(t, wantTexts[table], itWdbAllTexts(t, conn, table), "[%s] 复杂文本逐值不一致", table)
			}
		})
	}
}

// TestITWholeDbDataOnlyImportFailureNoResidue 仅数据流（无DDL）末尾失败：事务内已执行写入必须整体回滚。
//
// 为何只数据流适用：mysql 的 DROP/CREATE TABLE 会隐式提交当前事务（DDL 在 mysql 上不参事务），
// 含DDL的整库流无法做到“失败整体回滚”，其正确性保障为下例的“重跑幂等”。
// 数据流中每张表的注释头+BEGIN若漏过滤，后一张表的BEGIN会隐式提交前一张表的在途数据，
// 本断言即可抓到部分批次落库
func TestITWholeDbDataOnlyImportFailureNoResidue(t *testing.T) {
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			itWdbSetup(t, conn, 300)
			// 仅数据：无 DROP/CREATE，目标表保持存在，写入全部落在导入事务内
			script := itWdbDumpWholeDb(t, conn, node.dbType, false) + "\nINSERT INTO wdb_no_such_table VALUES (1);\n"
			for _, table := range itWdbTables {
				_, err := conn.Exec(fmt.Sprintf("DELETE FROM %s", conn.GetDialect().Quoter().QuoteIdent(table)))
				require.NoError(t, err)
			}

			app := &transfer.DbTransferAppImpl{}
			err := app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script))
			require.Error(t, err, "末尾坏语句应导致导入失败")

			for _, table := range itWdbTables {
				assert.Equal(t, int64(0), itWdbRowsSafe(t, conn, table),
					"[%s] 纯数据导入失败后不得残留部分批次数据", table)
			}
		})
	}
}

// TestITWholeDbFailedImportRerunIdempotent 含DDL的整库流失败后重跑：必须回到与备份前完全一致的状态。
//
// 导入侧不依赖“失败回滚”而依赖“表级 DROP 重建幂等”推翻残留，本用例验证该承诺在
// 四方言、多表单流、批次跨表边界场景下真实成立（既不得重复也不得丢失）
func TestITWholeDbFailedImportRerunIdempotent(t *testing.T) {
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			itWdbSetup(t, conn, 300)
			wantAgg, wantTexts := map[string]string{}, map[string][]string{}
			for _, table := range itWdbTables {
				wantAgg[table] = itWdbAgg(t, conn, table)
				wantTexts[table] = itWdbAllTexts(t, conn, table)
			}

			clean := itWdbDumpWholeDb(t, conn, node.dbType, true)
			app := &transfer.DbTransferAppImpl{}

			// 第一次：末尾坏语句→预期失败（mysql 上DDL已隐式提交，失败后残留部分数据属预期）
			itWdbDropAll(t, conn)
			require.Error(t, app.ImportDumpStream(context.Background(), 0, conn,
				strings.NewReader(clean+"\nINSERT INTO wdb_no_such_table VALUES (1);\n")), "末尾坏语句应导致导入失败")

			// 第二次：同一脚本完整重跑，必须与备份前逐值一致（残留被表级DROP重建自清理）
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(clean)),
				"失败后重跑应幂等成功")
			for _, table := range itWdbTables {
				assert.Equal(t, int64(300), itWdbRowsSafe(t, conn, table), "[%s] 重跑后行数应等于备份前行数（不得翻倍或缺失）", table)
				assert.Equal(t, wantAgg[table], itWdbAgg(t, conn, table), "[%s] 重跑后聚合快照不一致", table)
				assert.Equal(t, wantTexts[table], itWdbAllTexts(t, conn, table), "[%s] 重跑后复杂文本逐值不一致", table)
			}
		})
	}
}

// TestITWholeDbHeteroMigration 异构整库单流迁移：源库导出为目标方言SQL文本，导入另一方言真实库
func TestITWholeDbHeteroMigration(t *testing.T) {
	combos := []itPair{
		{itMysql, itPg}, {itMysql, itSqlite}, {itMysql, itMssql},
		{itPg, itMysql}, {itPg, itSqlite}, {itPg, itMssql},
		{itSqlite, itMysql}, {itSqlite, itPg}, {itSqlite, itMssql},
		// mssql作为异构源：SQL Server元数据（nvarchar/datetime2/varbinary/标识列）→其余方言的整库单流迁移
		{itMssql, itMysql}, {itMssql, itPg}, {itMssql, itSqlite},
	}
	for _, c := range combos {
		c := c
		t.Run(c.src.name+"->"+c.tgt.name, func(t *testing.T) {
			srcConn, tgtConn := c.src.conn(t), c.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()

			itWdbSetup(t, srcConn, 300)
			wantTexts := map[string][]string{}
			for _, table := range itWdbTables {
				wantTexts[table] = itWdbAllTexts(t, srcConn, table)
			}
			script := itWdbDumpWholeDb(t, srcConn, c.tgt.dbType, true)
			itWdbDropAll(t, tgtConn)

			app := &transfer.DbTransferAppImpl{}
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
				"异构整库导入失败, dump脚本:\n%s", itTruncate(script, 6000))

			for _, table := range itWdbTables {
				assert.Equal(t, int64(300), itWdbRowsSafe(t, tgtConn, table), "[%s] 行数不一致", table)
				assert.Equal(t, wantTexts[table], itWdbAllTexts(t, tgtConn, table), "[%s] 复杂文本逐值不一致", table)
				// 二级索引属表结构，必须随结构段一并迁移到异构目标库
				assert.Contains(t, itWdbIndexNames(t, tgtConn, table), itWdbIndexName(table),
					"[%s] 二级索引未迁移, 目标实际索引: %v", table, itWdbIndexNames(t, tgtConn, table))
			}

			itWdbDropAll(t, srcConn)
			itWdbDropAll(t, tgtConn)
		})
	}
}

// TestITWholeDbDataOnlyProductCarriesNoDDL 「仅数据」导出产物必须只含数据语句，且能恢复到已存在索引的表。
//
// 真实回归背景：索引段曾不受DumpDDL门控，DumpDDL:false的产物仍含 ALTER TABLE ... ADD INDEX，
// 使仅数据备份恢复到已存在索引的表直接失败（mysql: Error 1061 Duplicate key name）；
// 且mysql的ADD INDEX会隐式提交当前事务，破坏导入侧批级提交/失败回滚语义
func TestITWholeDbDataOnlyProductCarriesNoDDL(t *testing.T) {
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			itWdbSetup(t, conn, 300)
			// 非空转前提：源表确实带二级索引，否则“产物不含索引DDL”恒真而无鉴别力
			for _, table := range itWdbTables {
				require.Contains(t, itWdbIndexNames(t, conn, table), itWdbIndexName(table),
					"[%s] 前置索引未建立，本用例断言无效", table)
			}

			script := itWdbDumpWholeDb(t, conn, node.dbType, false)
			assert.NotContains(t, script, "-- Table Index", "仅数据产物不应含索引段")

			splitter := conn.GetDialect().GetSQLSplitter()
			dataStmts := 0
			for _, stmt := range itSplitStmts(t, conn, script) {
				head := sqlparser.NormalizeStmtText(splitter, stmt)
				if head == "" || head == "BEGIN" || head == "COMMIT" {
					continue // 注释头与方言事务包装语句
				}
				dataStmts++
				assert.True(t, strings.HasPrefix(head, "INSERT"), "仅数据产物含非DML语句: %s", itTruncate(head, 200))
			}
			assert.Positive(t, dataStmts, "仅数据产物未含任何INSERT，本用例断言无效")

			// 真实恢复：保留表与索引、仅清空数据后导入仅数据产物，必须成功且不重复
			quote := conn.GetDialect().Quoter().QuoteIdent
			for _, table := range itWdbTables {
				_, err := conn.Exec(fmt.Sprintf("DELETE FROM %s", quote(table)))
				require.NoError(t, err)
			}
			app := &transfer.DbTransferAppImpl{}
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)),
				"仅数据备份恢复到已存在索引的表失败, dump脚本:\n%s", itTruncate(script, 3000))
			for _, table := range itWdbTables {
				assert.Equal(t, int64(300), itWdbRowsSafe(t, conn, table), "[%s] 恢复后行数不一致", table)
				assert.Contains(t, itWdbIndexNames(t, conn, table), itWdbIndexName(table), "[%s] 索引意外丢失", table)
			}
		})
	}
}

// TestITWholeDbTwoPhaseMigrationWithIndex 两阶段迁移（结构阶段→数据阶段）在带二级索引的表上必须成功。
//
// 对应 transferTableDDL2Db + transferTableData2Db 的真实标志位组合：结构阶段建表并建索引，
// 数据阶段仅导入数据。若数据阶段产物重复携带索引DDL，第二阶段必然因索引重名失败
func TestITWholeDbTwoPhaseMigrationWithIndex(t *testing.T) {
	combos := []itPair{
		{itMysql, itMysql}, {itPg, itPg}, {itSqlite, itSqlite}, {itMssql, itMssql},
		{itMysql, itPg}, {itSqlite, itMysql}, {itMssql, itMysql},
	}
	for _, c := range combos {
		c := c
		t.Run(c.src.name+"->"+c.tgt.name, func(t *testing.T) {
			srcConn, tgtConn := c.src.conn(t), c.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()

			itWdbSetup(t, srcConn, 300)
			// 同构组合src/tgt为同一库，必须在删表前完成快照与两阶段导出
			wantTexts := map[string][]string{}
			for _, table := range itWdbTables {
				wantTexts[table] = itWdbAllTexts(t, srcConn, table)
			}
			ddlScript := itWdbDumpFlags(t, srcConn, c.tgt.dbType, true, false)
			dataScript := itWdbDumpFlags(t, srcConn, c.tgt.dbType, false, true)

			itWdbDropAll(t, tgtConn)
			app := &transfer.DbTransferAppImpl{}

			require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn, strings.NewReader(ddlScript)),
				"结构阶段导入失败, dump脚本:\n%s", itTruncate(ddlScript, 3000))
			for _, table := range itWdbTables {
				assert.Equal(t, int64(0), itWdbRowsSafe(t, tgtConn, table), "[%s] 结构阶段不应写入数据", table)
				assert.Contains(t, itWdbIndexNames(t, tgtConn, table), itWdbIndexName(table), "[%s] 结构阶段未迁移索引", table)
			}

			require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn, strings.NewReader(dataScript)),
				"数据阶段导入失败, dump脚本:\n%s", itTruncate(dataScript, 3000))
			for _, table := range itWdbTables {
				assert.Equal(t, int64(300), itWdbRowsSafe(t, tgtConn, table), "[%s] 两阶段迁移后行数不一致", table)
				assert.Equal(t, wantTexts[table], itWdbAllTexts(t, tgtConn, table), "[%s] 两阶段迁移后文本不一致", table)
			}
			itWdbDropAll(t, tgtConn)
		})
	}
}
