package itest

// 窄整型（smallint）主键表的主键分片迁移真实链路集成测试（mysql/pg/sqlite/mssql）。
//
// 为何必须有真实库用例：迁移并行度依赖 PlanTableShards 读取主键 MIN/MAX 并用
// dbi.ValToInt64 归一，而各驱动对窄整型的返值形态差异极大——本机实测 mysql/pg/mssql
// 对 SMALLINT 列的 MIN/MAX **均**返回 int16（mssql进一步为 TINYINT→int16、INT→int32、BIGINT→int64，
// sqlite则为string），而 ValToInt64 当时只识别 int64/int/int32/uint64/float64/[]byte/string；
// 归一失败后 PlanTableShards 按「空表/非数值」语义返回 nil，大表并行迁移**静默退化为整表单任务**
//（既无报错也无日志）。单测只能证明函数自身行为，「哪个驱动返回哪种Go形态」只能实测，
// 故此处按方言矩阵钉死。
//
// 运行：cd server && go test -tags it -count=1 -run TestITNarrowIntPrimaryKey ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"mayfly-go/internal/db/dbm/dbi"
)

const itNarrowPkTable = "it_narrow_pk_tbl"

// itNarrowPkDDL 各方言 smallint 主键建表列定义（非自增，显式写入id）
var itNarrowPkDDL = map[string]string{
	"mysql":    "(id SMALLINT PRIMARY KEY, val VARCHAR(64))",
	"postgres": "(id smallint PRIMARY KEY, val VARCHAR(64))",
	"sqlite":   "(id SMALLINT PRIMARY KEY, val TEXT)",
	"mssql":    "(id SMALLINT PRIMARY KEY, val NVARCHAR(64))",
}

func itNarrowPrepare(t *testing.T, conn *dbi.DbConn, rowCount int) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	ddl, ok := itNarrowPkDDL[string(conn.Info.Type)]
	require.True(t, ok, "未配置该方言的smallint主键建表语句: %s", conn.Info.Type)
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(itNarrowPkTable)))
	require.NoError(t, err, "[%s] 清理表失败", conn.Info.Type)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s %s", quote(itNarrowPkTable), ddl))
	require.NoError(t, err, "[%s] 建表失败", conn.Info.Type)
	if rowCount == 0 {
		return
	}
	// 分批写入，避免单条INSERT超出服务端包上限
	for start := 1; start <= rowCount; start += 200 {
		end := min(start+199, rowCount)
		values := make([]string, 0, end-start+1)
		for i := start; i <= end; i++ {
			values = append(values, fmt.Sprintf("(%d, 'v%03d')", i, i))
		}
		_, err = conn.Exec(fmt.Sprintf("INSERT INTO %s (id, val) VALUES %s", quote(itNarrowPkTable), strings.Join(values, ",")))
		require.NoError(t, err, "[%s] 写入行失败", conn.Info.Type)
	}
}

func itNarrowAgg(t *testing.T, conn *dbi.DbConn, expr string) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT %s AS agg FROM %s", expr, quote(itNarrowPkTable)))
	require.NoError(t, err, "[%s] 查询聚合失败: %s", conn.Info.Type, expr)
	require.Len(t, rows, 1)
	agg, ok := dbi.ValToInt64(rows[0]["agg"])
	require.True(t, ok, "[%s] 聚合值形态异常: %T", conn.Info.Type, rows[0]["agg"])
	return agg
}

// TestITNarrowIntPrimaryKeySharding smallint主键表必须能真正切出多个分片，且分片并行导入后
// 行集合与源表完全一致（分片边界错一处即表现为丢行或重复）
func TestITNarrowIntPrimaryKeySharding(t *testing.T) {
	const (
		rows        = 200 // smallint范围内，且200/40 → 5个分片
		shardRows   = 40
		concurrency = 4
	)
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			origTargetRows := dbi.ShardTargetRows
			dbi.ShardTargetRows = shardRows
			defer func() { dbi.ShardTargetRows = origTargetRows }()

			itNarrowPrepare(t, conn, rows)
			defer func() {
				quote := conn.GetDialect().Quoter().QuoteIdent
				_, _ = conn.Exec("DROP TABLE IF EXISTS " + quote(itNarrowPkTable))
			}()

			// 元数据必须把smallint识别为可分片整型主键，否则本用例失去鉴别力
			columns, err := conn.GetMetadata().GetColumns(itNarrowPkTable)
			require.NoError(t, err, "[%s] 查询列元数据失败", node.name)
			require.Equal(t, "id", dbi.DetectIntPrimaryKey(columns),
				"[%s] smallint主键应判为可分片（元数据类型=%v）", node.name, columns)

			app := &transfer.DbTransferAppImpl{}
			wheres := app.PlanTableShards(context.Background(), 0, conn, itNarrowPkTable, rows)
			require.GreaterOrEqual(t, len(wheres), 3,
				"[%s] smallint主键表未切出分片：MIN/MAX返值归一失败，并行迁移已静默退化为单任务", node.name)

			ddlScript := itDumpTable(t, conn, itNarrowPkTable, node.dbType, true, false, "")
			shards := make([]string, 0, len(wheres))
			for _, where := range wheres {
				shard := itDumpTable(t, conn, itNarrowPkTable, node.dbType, false, true, where)
				require.NotContains(t, strings.ToUpper(shard), "CREATE TABLE", "[%s] 分片段产物不应含DDL", node.name)
				require.Contains(t, strings.ToUpper(shard), "INSERT INTO", "[%s] 分片段产物应含数据语句", node.name)
				shards = append(shards, shard)
			}

			require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(ddlScript)),
				"[%s] 结构阶段导入失败", node.name)
			require.Equal(t, int64(0), itNarrowAgg(t, conn, "COUNT(*)"), "[%s] 结构阶段不应携带数据", node.name)

			eg := new(errgroup.Group)
			eg.SetLimit(concurrency)
			for i, shard := range shards {
				i, shard := i, shard
				eg.Go(func() error {
					if err := app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(shard)); err != nil {
						return fmt.Errorf("分片%d导入失败: %w\n%s", i+1, err, itTruncate(shard, 2000))
					}
					return nil
				})
			}
			require.NoError(t, eg.Wait(), "[%s] 分片并行导入失败", node.name)

			assert.Equal(t, int64(rows), itNarrowAgg(t, conn, "COUNT(*)"), "[%s] 并行导入后行数不一致（丢行或重复）", node.name)
			assert.Equal(t, int64(rows), itNarrowAgg(t, conn, "COUNT(DISTINCT id)"), "[%s] id存在重叠或空洞", node.name)
			assert.Equal(t, int64(1), itNarrowAgg(t, conn, "MIN(id)"), "[%s] 最小id与源表不符", node.name)
			assert.Equal(t, int64(rows), itNarrowAgg(t, conn, "MAX(id)"), "[%s] 最大id与源表不符", node.name)
		})
	}
}

// TestITNarrowIntPrimaryKeyEmptyTable 空表（MIN/MAX为NULL）不得分片，且不得因取值失败而报错：
// 该场景归一化必须走「取不到值→整表迁移」路径
func TestITNarrowIntPrimaryKeyEmptyTable(t *testing.T) {
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			itNarrowPrepare(t, conn, 0)
			defer func() {
				quote := conn.GetDialect().Quoter().QuoteIdent
				_, _ = conn.Exec("DROP TABLE IF EXISTS " + quote(itNarrowPkTable))
			}()

			app := &transfer.DbTransferAppImpl{}
			assert.Nil(t, app.PlanTableShards(context.Background(), 0, conn, itNarrowPkTable, 0),
				"[%s] 空表不应分片", node.name)
		})
	}
}
