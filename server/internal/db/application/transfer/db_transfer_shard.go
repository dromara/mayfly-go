package transfer

import (
	"context"
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
)

// 迁移并行度边界：数据导入工作池大小
const (
	// DefaultTransferConcurrency 默认并行度
	DefaultTransferConcurrency = 4
	// MinTransferConcurrency 最小并行度
	MinTransferConcurrency = 1
	// MaxTransferConcurrency 最大并行度（防止过大并行拖垮源/目标库）
	MaxTransferConcurrency = 16
)

// normalizeConcurrency 归一化迁移并行度：0→默认值，越界钳制到[1,16]
func normalizeConcurrency(c int) int {
	switch {
	case c <= 0:
		return DefaultTransferConcurrency
	case c < MinTransferConcurrency:
		return MinTransferConcurrency
	case c > MaxTransferConcurrency:
		return MaxTransferConcurrency
	default:
		return c
	}
}

// PlanTableShards 规划表的主键范围分片，返回每片的数据过滤条件（空字符串=整表无过滤）。
//
// 仅当表存在**真实单列整型主键**时分片（见dbi.DetectIntPrimaryKey）；否则返回nil，
// 调用方应整表单任务迁移。规划失败（元数据查询异常等）降级为整表迁移，不阻断迁移流程。
func (app *DbTransferAppImpl) PlanTableShards(ctx context.Context, logId uint64, srcConn *dbi.DbConn, tableName string, tableRows int) []string {
	columns, err := srcConn.GetMetadata().GetColumns(tableName)
	if err != nil {
		app.Log(ctx, logId, fmt.Sprintf("plan shards: get columns of table [%s] failed: %s, fallback to whole-table transfer", tableName, err.Error()))
		return nil
	}

	pk := dbi.DetectIntPrimaryKey(columns)
	if pk == "" {
		return nil
	}

	// 查询主键范围（空表MIN/MAX为NULL → 不分片）
	quote := srcConn.GetDialect().Quoter().QuoteIdent
	minMaxSql := fmt.Sprintf("SELECT MIN(%s) AS mn, MAX(%s) AS mx FROM %s", quote(pk), quote(pk), quote(tableName))
	_, rows, err := srcConn.Query(minMaxSql)
	if err != nil || len(rows) == 0 {
		app.Log(ctx, logId, fmt.Sprintf("plan shards: query pk range of table [%s] failed: %v, fallback to whole-table transfer", tableName, err))
		return nil
	}
	pkMin, ok1 := dbi.ValToInt64(rows[0]["mn"])
	pkMax, ok2 := dbi.ValToInt64(rows[0]["mx"])
	if !ok1 || !ok2 {
		// 空表或非数值返回形态：整表迁移（可能0行，dump自然无数据）
		return nil
	}

	shards := dbi.PlanShards(pk, pkMin, pkMax, tableRows, quote)
	if shards == nil {
		return nil
	}
	wheres := make([]string, 0, len(shards))
	for _, s := range shards {
		wheres = append(wheres, s.Where)
	}
	app.Log(ctx, logId, fmt.Sprintf("table [%s] pk range [%d, %d], planned %d shard(s)", tableName, pkMin, pkMax, len(wheres)))
	return wheres
}
