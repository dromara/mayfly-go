package itest

// 特殊字符表名的迁移链路真实集成测试。
//
// 表名会流经：元数据查询（GetTables/GetColumns）→ dump DDL/INSERT生成（目标方言quote）→
// 脚本切割（目标方言splitter，标识符内含分号/注释符/引号）→ 导入执行 → 校验器按表名回查元数据。
// 任一环节quote或切割状态机处理不当，都会导致建表失败、语句错切或数据丢失。
//
// 覆盖表名形态：中文、空格、单引号、双引号、反引号、分号、行注释符--、mysql哈希注释符#、
// 块注释符/*、LIKE通配符%与_、反斜杠、以及混合形态。
//
// 运行：cd server && go test -tags it -count=1 -run TestITSpecialTableNames ./internal/db/application/

import (
	"mayfly-go/internal/db/application/transfer"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// itSpecialTableNames 特殊表名后缀样本（desc用于失败信息说明）
var itSpecialTableNames = []struct{ suffix, desc string }{
	{"中文表", "中文"},
	{"空格 表", "含空格"},
	{"单'引号", "含单引号"},
	{`双"引号`, "含双引号"},
	{"反`引号", "含反引号"},
	{"分;号", "含分号（切割器最易错切）"},
	{"横线--注释", "含行注释符"},
	{"哈希#注释", "含mysql哈希注释符"},
	{"块/**/注释", "含块注释符"},
	{"通配%_", "含LIKE通配符"},
	{"反\\斜杠", "含反斜杠"},
	{";--'\"`#/*", "全混合高危形态"},
}

// TestITSpecialTableNamesAcrossDialects 特殊表名三方言迁移链路
func TestITSpecialTableNamesAcrossDialects(t *testing.T) {
	pairs := []itPair{
		{itMysql, itPg}, {itPg, itMysql}, {itSqlite, itPg},
	}

	for _, p := range pairs {
		t.Run(p.src.name+"->"+p.tgt.name, func(t *testing.T) {
			srcConn, tgtConn := p.src.conn(t), p.tgt.conn(t)
			defer srcConn.Close()
			defer tgtConn.Close()
			tgtQuote := tgtConn.GetDialect().Quoter().QuoteIdent

			for _, sn := range itSpecialTableNames {
				// 统一前缀便于识别测试表，且控制在各方言标识符长度限制内
				table := fmt.Sprintf("it_cxname_%s_%s", p.src.name, sn.suffix)
				t.Run(sn.desc, func(t *testing.T) {
					// 源侧清理：上一次异常中断可能残留同名表
					_, err := srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcConn.GetDialect().Quoter().QuoteIdent(table)))
					require.NoError(t, err)
					itCreateComplexTable(t, srcConn, table)
					wantRows := itInsertComplexRows(t, srcConn, table)

					script := itDumpTable(t, srcConn, table, p.tgt.dbType, true, true, "")
					app := &transfer.DbTransferAppImpl{}
					require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
						"导入失败, dump脚本:\n%s", itTruncate(script, 4000))

					// 目标侧真实按同名表回读（不经过任何名称重写），验证quote与切割全链路保真
					_, rows, err := tgtConn.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", tgtQuote(table)))
					require.NoError(t, err, "特殊表名目标表查询失败")
					require.Len(t, rows, 1)
					cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
					require.True(t, ok, "count应为数值: %#v", rows[0]["cnt"])
					assert.Equal(t, int64(wantRows), cnt, "特殊表名迁移行数不应丢失或重复")

					// 内容逐列比对（含数据列的复杂字符串）
					srcRows := itReadRowsRaw(t, srcConn, table)
					tgtRows := itReadRowsRaw(t, tgtConn, table)
					require.Len(t, tgtRows, wantRows)
					for i := 0; i < wantRows; i++ {
						for _, col := range itTextColumns {
							assert.Equal(t, itTextAt(srcRows[i], col), itTextAt(tgtRows[i], col),
								"第%d行列[%s]迁移失真", i+1, col)
						}
					}

					// 产品校验器需能按特殊表名回查元数据并完成比对
					res := app.VerifyTable(context.Background(), srcConn, tgtConn, table)
					require.Empty(t, res.Err, "校验器失败: %s", res.Err)
					assert.True(t, res.CountMatch, "校验器count应一致")
					assert.Empty(t, res.MismatchPk, "校验器不应误报: %v", res.MismatchPk)

					// 清理，避免残留特殊表名影响人工排查
					_, _ = srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcConn.GetDialect().Quoter().QuoteIdent(table)))
					_, _ = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tgtQuote(table)))
				})
			}
		})
	}
}

// TestITSpecialTableNamesShardMigrate 特殊表名 + 大表分片：TableFilter按表名匹配where条件
func TestITSpecialTableNamesShardMigrate(t *testing.T) {
	srcConn, tgtConn := itMysql.conn(t), itPg.conn(t)
	defer srcConn.Close()
	defer tgtConn.Close()
	srcQuote, tgtQuote := srcConn.GetDialect().Quoter().QuoteIdent, tgtConn.GetDialect().Quoter().QuoteIdent

	table := "it_cxshard_中文;分号--x"
	_, err := srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcQuote(table)))
	require.NoError(t, err)
	itCreateShardComplexTable(t, srcConn, table)
	itInsertShardComplexRows(t, srcConn, table, shardCxRows)

	origTargetRows := dbi.ShardTargetRows
	dbi.ShardTargetRows = 1000
	defer func() { dbi.ShardTargetRows = origTargetRows }()

	app := &transfer.DbTransferAppImpl{}
	wheres := app.PlanTableShards(context.Background(), 0, srcConn, table, shardCxRows)
	require.Len(t, wheres, 3, "特殊表名不应影响分片规划")

	require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn,
		strings.NewReader(itDumpTable(t, srcConn, table, itPg.dbType, true, false, ""))))
	for _, w := range wheres {
		// 串行导入：逐分片验证TableFilter按特殊表名能正确命中过滤条件
		script := itDumpTable(t, srcConn, table, itPg.dbType, false, true, w)
		require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
			"分片[%s]导入失败:\n%s", w, itTruncate(script, 2000))
	}

	_, rows, err := tgtConn.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", tgtQuote(table)))
	require.NoError(t, err)
	cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
	require.True(t, ok)
	assert.Equal(t, int64(shardCxRows), cnt, "特殊表名分片迁移不应丢行或重复")

	res := app.VerifyTable(context.Background(), srcConn, tgtConn, table)
	require.Empty(t, res.Err)
	assert.True(t, res.CountMatch)
	assert.Empty(t, res.MismatchPk, "特殊表名分片迁移后校验器不应误报: %v", res.MismatchPk)

	_, _ = srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcQuote(table)))
	_, _ = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tgtQuote(table)))
}
