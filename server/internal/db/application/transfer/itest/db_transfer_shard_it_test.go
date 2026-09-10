package itest

// 大表主键分片迁移真实链路集成测试：
// mysql源（3000行整型主键表）→ 分片规划（调小ShardTargetRows验证多分片）→ pg目标并行导入，
// 验证分片where条件无缝覆盖、行数一致、无重复无丢失、内容抽样比对。
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/application/

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

const shardItRows = 3000

// parseShardWhere 解析分片where条件中的主键闭区间（quote形式的前后缀由方言决定，先剥离列引用）
func parseShardWhere(t *testing.T, where string) (lo, hi int64) {
	t.Helper()
	// 兼容 `id`/id/"id" 引用形式
	normalized := strings.NewReplacer("`", "", "\"", "").Replace(where)
	n, err := fmt.Sscanf(normalized, "id >= %d AND id <= %d", &lo, &hi)
	require.NoError(t, err, "where格式异常: %s", where)
	require.Equal(t, 2, n)
	return lo, hi
}

func dbValToStr(v any) string {
	switch x := v.(type) {
	case nil:
		return "<nil>"
	case []byte:
		return string(x)
	case string:
		return x
	default:
		return fmt.Sprintf("%v", x)
	}
}

// TestITShardPlanAndMigrate_MysqlToPg mysql→pg分片迁移全链路
func TestITShardPlanAndMigrate_MysqlToPg(t *testing.T) {
	srcConn := transferTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	defer srcConn.Close()
	tgtConn := transferTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	defer tgtConn.Close()

	srcTable, tgtTable := "it_shard_src", "it_shard_tgt"
	srcQuote := srcConn.GetDialect().Quoter().Quote
	tgtQuote := tgtConn.GetDialect().Quoter().Quote

	// 准备源表：3000行整型主键数据（复用导入链路建表插数）
	_, err := srcConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", srcQuote(srcTable)))
	require.NoError(t, err)
	require.NoError(t, (&transfer.DbTransferAppImpl{}).ImportDumpStream(context.Background(), 0, srcConn,
		strings.NewReader(buildDumpScript(srcQuote, srcTable, shardItRows))))

	// 调小分片目标行数：3000行/1000行每片 → 3片（验证多分片路径）
	origTargetRows := dbi.ShardTargetRows
	dbi.ShardTargetRows = 1000
	defer func() { dbi.ShardTargetRows = origTargetRows }()

	app := &transfer.DbTransferAppImpl{}
	wheres := app.PlanTableShards(context.Background(), 0, srcConn, srcTable, shardItRows)
	require.Len(t, wheres, 3, "3000行、每片1000行应规划为3片")

	// 分片闭区间无缝覆盖[1, 3000]，无重叠
	require.Equal(t, int64(1), func() int64 { lo, _ := parseShardWhere(t, wheres[0]); return lo }(), "首片应从min开始")
	prevHi := int64(0)
	for i, w := range wheres {
		lo, hi := parseShardWhere(t, w)
		if i > 0 {
			assert.Equal(t, prevHi+1, lo, "分片%d与前一档应无缝衔接", i)
		}
		assert.LessOrEqual(t, lo, hi)
		prevHi = hi
	}
	assert.Equal(t, int64(shardItRows), prevHi, "末片应覆盖到max")

	// 目标表
	_, err = tgtConn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tgtQuote(tgtTable)))
	require.NoError(t, err)
	_, err = tgtConn.Exec(fmt.Sprintf("CREATE TABLE %s (id int PRIMARY KEY, val varchar(64))", tgtQuote(tgtTable)))
	require.NoError(t, err)

	// 分片并行导入（模拟阶段2工作池：SetLimit并发 + 各分片独立dump查询→批量导入）
	var eg errgroup.Group
	eg.SetLimit(2)
	for _, w := range wheres {
		w := w
		eg.Go(func() error {
			var shardRows []map[string]any
			if _, err := srcConn.WalkQueryRows(context.Background(),
				fmt.Sprintf("SELECT * FROM %s WHERE %s", srcQuote(srcTable), w),
				func(row map[string]any, _ []*dbi.QueryColumn) error {
					shardRows = append(shardRows, row)
					return nil
				}); err != nil {
				return err
			}

			tx, err := tgtConn.Begin()
			if err != nil {
				return err
			}
			for _, row := range shardRows {
				insSql := fmt.Sprintf("INSERT INTO %s (id, val) VALUES (%v, '%s')", tgtQuote(tgtTable), row["id"], dbValToStr(row["val"]))
				if _, err := tgtConn.TxExec(tx, insSql); err != nil {
					dbi.RollbackTx(tx)
					return err
				}
			}
			return dbi.CommitTargetTx(tgtConn, tx)
		})
	}
	require.NoError(t, eg.Wait())

	// 行数一致
	assert.Equal(t, shardItRows, countRows(t, tgtConn, tgtQuote, tgtTable))

	// 无重复无丢失：distinct id数=3000且id范围[1,3000]
	_, statRows, err := tgtConn.Query(fmt.Sprintf(
		"SELECT COUNT(DISTINCT id) AS cnt, MIN(id) AS mn, MAX(id) AS mx FROM %s", tgtQuote(tgtTable)))
	require.NoError(t, err)
	cnt, ok1 := dbi.ValToInt64(statRows[0]["cnt"])
	mn, ok2 := dbi.ValToInt64(statRows[0]["mn"])
	mx, ok3 := dbi.ValToInt64(statRows[0]["mx"])
	require.True(t, ok1 && ok2 && ok3, "统计结果应为数值形态: %#v", statRows[0])
	assert.Equal(t, int64(shardItRows), cnt, "分片导入不应有重复或丢失")
	assert.Equal(t, int64(1), mn)
	assert.Equal(t, int64(shardItRows), mx)

	// 内容抽样比对（首/中/尾）
	for _, id := range []int{1, shardItRows / 2, shardItRows} {
		_, sampleRows, err := tgtConn.Query(fmt.Sprintf("SELECT val FROM %s WHERE id = %d", tgtQuote(tgtTable), id))
		require.NoError(t, err)
		require.Len(t, sampleRows, 1)
		assert.Equal(t, fmt.Sprintf("v%d", id), dbValToStr(sampleRows[0]["val"]), "id=%d内容应精确还原", id)
	}
}

// TestITPlanShards_PgMetadata pg源元数据分片判定：int4主键可识别分片
func TestITPlanShards_PgMetadata(t *testing.T) {
	conn := transferTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	defer conn.Close()
	quote := conn.GetDialect().Quoter().Quote

	table := "it_shard_pg_meta"
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id serial PRIMARY KEY, val varchar(32))", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("INSERT INTO %s (val) VALUES ('a'), ('b'), ('c')", quote(table)))
	require.NoError(t, err)

	origTargetRows := dbi.ShardTargetRows
	dbi.ShardTargetRows = 2 // 3行/2行每片 → 2片
	defer func() { dbi.ShardTargetRows = origTargetRows }()

	app := &transfer.DbTransferAppImpl{}
	wheres := app.PlanTableShards(context.Background(), 0, conn, table, 3)
	require.Len(t, wheres, 2, "pg serial主键应识别为可分片")

	// 无联合主键/varchar主键表 → 不分片
	noPkTable := "it_shard_pg_nopk"
	_, err = conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(noPkTable)))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (code varchar(32), val varchar(32))", quote(noPkTable)))
	require.NoError(t, err)

	wheres = app.PlanTableShards(context.Background(), 0, conn, noPkTable, 100)
	assert.Nil(t, wheres, "无主键表不应分片")
}
