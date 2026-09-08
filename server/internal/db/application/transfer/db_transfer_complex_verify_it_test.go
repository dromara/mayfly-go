//go:build it

package transfer

// 复杂字符串在「产品校验器」与「大表主键分片迁移」两条真实链路上的集成测试。
//
// 与 TestITComplexStringsAcrossDialects（dump→导入→回读字节级比对）互补，本文件驱动产品侧函数：
//   - verifyTable：跨方言下对含引号/反斜杠/换行/JSON/二进制/NULL的表做count+头尾抽样比对，
//     既验证不误报（数值标度与时间/二进制形态差异经规范化与数值语义判等后一致），
//     也验证不漏报（文本列被篡改为另一复杂值、NULL与空串混淆必须精确检出）。
//   - planTableShards + dump(TableFilter) + importDumpStream：数千行复杂文本大表分片并行迁移，
//     全量逐行逐列比对，确保分片边界处语句切割与批量导入不丢行、不改值、不重复。
//
// 运行：cd server && go test -tags it -count=1 -run TestITComplex ./internal/db/application/

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

// itDumpTable 驱动真实dump生成脚本：可选DDL/数据、可选分片where条件
func itDumpTable(t *testing.T, srcConn *dbi.DbConn, table string, targetDbType dbi.DbType, dumpDDL, dumpData bool, where string) string {
	t.Helper()
	var buf strings.Builder
	dump := &dto.DumpDb{
		DbName:       srcConn.Info.Database,
		Tables:       []string{table},
		DumpDDL:      dumpDDL,
		DumpData:     dumpData,
		TargetDbType: targetDbType,
		Writer:       &buf,
	}
	if where != "" {
		dump.TableFilter = map[string]string{table: where}
	}
	require.NoError(t, DumpDbScript(context.Background(), srcConn, dump), "dump失败")
	script := buf.String()
	require.NotEmpty(t, script)
	return script
}

// itExecParam 参数化执行DML（不手写SQL字面量转义，确保写入的是原始值）
func itExecParam(t *testing.T, conn *dbi.DbConn, sqlFormat string, table string, args ...any) {
	t.Helper()
	sqlStr := fmt.Sprintf(sqlFormat, conn.GetDialect().Quoter().QuoteIdent(table))
	_, err := conn.Exec(sqlStr, args...)
	require.NoError(t, err, "参数化执行失败: %s", sqlStr)
}

// itSetColumnParam 按主键更新单列（参数化）
func itSetColumnParam(t *testing.T, conn *dbi.DbConn, table, column string, id int64, val any) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	setPh, idPh := itPlaceholder(conn, 1), itPlaceholder(conn, 2)
	itExecParam(t, conn, fmt.Sprintf("UPDATE %%s SET %s = %s WHERE id = %s", quote(column), setPh, idPh),
		table, val, id)
}

// itSetupComplexPair 建源表+写入复杂样本+真实dump导入目标库（同表名，符合真实迁移语义）
func itSetupComplexPair(t *testing.T, src, tgt itDialectNode, table string) (srcConn, tgtConn *dbi.DbConn, wantRows int) {
	t.Helper()
	srcConn, tgtConn = src.conn(t), tgt.conn(t)
	itCreateComplexTable(t, srcConn, table)
	wantRows = itInsertComplexRows(t, srcConn, table)

	script := itDumpTable(t, srcConn, table, tgt.dbType, true, true, "")
	require.NoError(t, (&DbTransferAppImpl{}).importDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
		"导入失败, dump脚本:\n%s", itTruncate(script, 4000))
	return
}

// itVerifyComplexPairs 参与校验器/分片链路互测的方言有序对
var itVerifyComplexPairs = []struct{ src, tgt itDialectNode }{
	{itMysql, itPg}, {itMysql, itSqlite},
	{itPg, itMysql}, {itPg, itSqlite},
	{itSqlite, itMysql}, {itSqlite, itPg},
}

// TestITComplexStringsVerifyAcrossDialects 迁移后的复杂值表交由产品校验器比对：
// 一致时必须零误报，篡改复杂值/NULL混淆时必须精确检出
func TestITComplexStringsVerifyAcrossDialects(t *testing.T) {
	for _, p := range itVerifyComplexPairs {
		t.Run(p.src.name+"->"+p.tgt.name, func(t *testing.T) {
			table := fmt.Sprintf("it_cxv_%s_%s", p.src.name, p.tgt.name)
			srcConn, tgtConn, wantRows := itSetupComplexPair(t, p.src, p.tgt, table)
			defer srcConn.Close()
			defer tgtConn.Close()

			app := &DbTransferAppImpl{}
			res := app.verifyTable(context.Background(), srcConn, tgtConn, table)
			require.Empty(t, res.Err, "校验器执行失败: %s", res.Err)
			require.Empty(t, res.SampleErr, "单列主键表不应跳过内容抽样: %s", res.SampleErr)
			assert.True(t, res.CountMatch, "迁移后两侧count应一致")
			assert.Equal(t, int64(wantRows), res.SrcCount)
			assert.Equal(t, int64(wantRows), res.TargetCount)
			assert.Equal(t, wantRows, res.Sampled, "抽样行数应按主键去重后等于表行数")
			assert.Empty(t, res.MismatchPk, "跨方言复杂值（含数值标度/时间/二进制形态差异）不应误报: %v", res.MismatchPk)

			// 篡改1：文本列改为另一含引号/反斜杠/分号的复杂值（与真实值长度接近，仅内容不同）
			itSetColumnParam(t, tgtConn, table, "v_text", 3, `tampered;'"\\\;--`)
			// 篡改2：NULL列改为空串——必须与NULL区分检出（规范化串"<nil>"与""不同形态）
			itSetColumnParam(t, tgtConn, table, "v_null", 1, "")
			// 篡改3：二进制列改一个字节
			itSetColumnParam(t, tgtConn, table, "v_blob", 5, append(append([]byte{}, itComplexBlob...), 'X'))

			res2 := app.verifyTable(context.Background(), srcConn, tgtConn, table)
			require.Empty(t, res2.Err)
			assert.True(t, res2.CountMatch, "仅改值不删行，count仍应一致")
			assert.ElementsMatch(t, []string{"1", "3", "5"}, res2.MismatchPk,
				"被篡改行必须按主键精确检出，不得漏报也不得波及其他行")
		})
	}
}

// ---------------- 大表复杂值分片迁移 ----------------

const shardCxRows = 2400

// itCreateShardComplexTable 分片测试表：整型主键 + 复杂文本 + 二进制 + 数值
func itCreateShardComplexTable(t *testing.T, conn *dbi.DbConn, table string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	var ddl string
	switch conn.Info.Type {
	case "mysql":
		ddl = fmt.Sprintf("CREATE TABLE %s (id int PRIMARY KEY, v_text text, v_blob blob, v_num decimal(20,6)) DEFAULT CHARSET=utf8mb4", quote(table))
	case "postgres":
		ddl = fmt.Sprintf("CREATE TABLE %s (id int PRIMARY KEY, v_text text, v_blob bytea, v_num numeric(20,6))", quote(table))
	case "sqlite":
		ddl = fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, v_text TEXT, v_blob BLOB, v_num NUMERIC)", quote(table))
	default:
		t.Fatalf("unsupported dialect %s", conn.Info.Type)
	}
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(ddl)
	require.NoError(t, err, "建表失败")
}

// itInsertShardComplexRows 参数化批量写入复杂值（每语句50行，避免手写转义）
func itInsertShardComplexRows(t *testing.T, conn *dbi.DbConn, table string, total int) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	const chunk = 50
	cols := []string{"id", "v_text", "v_blob", "v_num"}
	for start := 1; start <= total; start += chunk {
		end := min(start+chunk-1, total)
		var sb strings.Builder
		fmt.Fprintf(&sb, "INSERT INTO %s (%s) VALUES ", quote(table), strings.Join(cols, ", "))
		args := make([]any, 0, (end-start+1)*len(cols))
		for i, id := 0, start; id <= end; id, i = id+1, i+1 {
			if i > 0 {
				sb.WriteString(", ")
			}
			ph := make([]string, 0, len(cols))
			for j := range cols {
				ph = append(ph, itPlaceholder(conn, j+1+i*len(cols)))
			}
			fmt.Fprintf(&sb, "(%s)", strings.Join(ph, ", "))
			args = append(args, int64(id), itComplexTexts[(id-1)%len(itComplexTexts)], itComplexBlob, "123456.456000")
		}
		_, err := conn.Exec(sb.String(), args...)
		require.NoError(t, err, "批量写入%d~%d行失败", start, end)
	}
}

// TestITShardComplexValuesMigrate 数千行复杂值大表：真实分片规划 → 分片dump → 并行导入 → 全量比对
func TestITShardComplexValuesMigrate(t *testing.T) {
	pairs := []struct {
		src, tgt itDialectNode
		imports  int // 并发导入数（sqlite单写者故串行）
	}{
		{itMysql, itPg, 2},
		{itPg, itSqlite, 1},
	}
	for _, p := range pairs {
		t.Run(p.src.name+"->"+p.tgt.name, func(t *testing.T) {
			table := fmt.Sprintf("it_shardcx_%s_%s", p.src.name, p.tgt.name)
			srcConn, tgtConn := p.src.conn(t), p.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()

			itCreateShardComplexTable(t, srcConn, table)
			itInsertShardComplexRows(t, srcConn, table, shardCxRows)

			origTargetRows := dbi.ShardTargetRows
			dbi.ShardTargetRows = 1000 // 2400行/每片1000行 → 3片
			defer func() { dbi.ShardTargetRows = origTargetRows }()

			app := &DbTransferAppImpl{}
			wheres := app.planTableShards(context.Background(), 0, srcConn, table, shardCxRows)
			require.Len(t, wheres, 3, "2400行、每片1000行应规划为3片")

			// 阶段1：表结构迁移（真实dump DDL，含drop重建）
			require.NoError(t, app.importDumpStream(context.Background(), 0, tgtConn,
				strings.NewReader(itDumpTable(t, srcConn, table, p.tgt.dbType, true, false, ""))))

			// 阶段2：分片并行导入（与产品Run一致：各分片dump仅数据 + 真实批量导入）
			var eg errgroup.Group
			eg.SetLimit(p.imports)
			for _, w := range wheres {
				w := w
				eg.Go(func() error {
					script := itDumpTable(t, srcConn, table, p.tgt.dbType, false, true, w)
					return app.importDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script))
				})
			}
			require.NoError(t, eg.Wait())

			// 无丢失无重复
			tgtQuote := tgtConn.GetDialect().Quoter().QuoteIdent
			_, statRows, err := tgtConn.Query(fmt.Sprintf(
				"SELECT COUNT(*) AS cnt, COUNT(DISTINCT id) AS dc, MIN(id) AS mn, MAX(id) AS mx FROM %s", tgtQuote(table)))
			require.NoError(t, err)
			require.Len(t, statRows, 1)
			for _, key := range []string{"cnt", "dc"} {
				cnt, ok := dbi.ValToInt64(statRows[0][key])
				require.True(t, ok, "%s应为数值: %#v", key, statRows[0][key])
				assert.Equal(t, int64(shardCxRows), cnt, "%s分片导入不应丢行或重复", key)
			}
			mn, ok1 := dbi.ValToInt64(statRows[0]["mn"])
			mx, ok2 := dbi.ValToInt64(statRows[0]["mx"])
			require.True(t, ok1 && ok2)
			assert.Equal(t, int64(1), mn)
			assert.Equal(t, int64(shardCxRows), mx)

			// 全量逐行逐列比对：文本/二进制字节级保真，数值列按数值语义
			srcRows := itReadRowsRaw(t, srcConn, table)
			tgtRows := itReadRowsRaw(t, tgtConn, table)
			require.Len(t, srcRows, shardCxRows)
			require.Len(t, tgtRows, shardCxRows, "目标库回读行数不符")
			sampleOf := func(i int) string { return itTruncate(itComplexTexts[i%len(itComplexTexts)], 60) }
			for i := 0; i < shardCxRows; i++ {
				for _, col := range []string{"id", "v_text", "v_blob"} {
					assert.Equal(t, itTextAt(srcRows[i], col), itTextAt(tgtRows[i], col),
						"第%d行(%q)列[%s]分片迁移失真, 源=%q 目标=%q", i+1, sampleOf(i), col,
						itTruncate(itTextAt(srcRows[i], col), 80), itTruncate(itTextAt(tgtRows[i], col), 80))
				}
				assert.True(t, dbi.CanonicalNumericEqual(srcRows[i]["v_num"], tgtRows[i]["v_num"]),
					"第%d行列[v_num]分片迁移失真, 源=%q 目标=%q", i+1,
					dbi.CanonicalValue(srcRows[i]["v_num"]), dbi.CanonicalValue(tgtRows[i]["v_num"]))
			}

			// 产品校验器对分片迁移结果应零误报
			res := app.verifyTable(context.Background(), srcConn, tgtConn, table)
			require.Empty(t, res.Err)
			assert.True(t, res.CountMatch)
			assert.Empty(t, res.MismatchPk, "分片迁移后校验器不应误报: %v", res.MismatchPk)
		})
	}
}
