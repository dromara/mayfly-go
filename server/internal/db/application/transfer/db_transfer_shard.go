package transfer

import (
	"context"
	"fmt"

	"mayfly-go/internal/db/application/shard"
	"mayfly-go/internal/db/dbm/dbi"
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
// 区间数学与探测均在 shard 包（迁移与数据同步共用一份，见该包注释），本函数只负责
// 把它接进迁移日志语义：不可分片返回 nil（调用方整表单任务），探测失败也降级为整表迁移、
// 不阻断迁移流程——分片是加速手段，不是正确性前提。
func (app *DbTransferAppImpl) PlanTableShards(ctx context.Context, logId uint64, srcConn *dbi.DbConn, tableName string, tableRows int) []string {
	shards, err := shard.PlanTable(ctx, srcConn, tableName, tableRows, 0)
	if err != nil {
		app.Log(ctx, logId, fmt.Sprintf("plan shards: %s, fallback to whole-table transfer", err.Error()))
		return nil
	}
	if shards == nil {
		return nil
	}
	wheres := make([]string, 0, len(shards))
	for _, s := range shards {
		wheres = append(wheres, s.Where)
	}
	if len(wheres) > 1 {
		app.Log(ctx, logId, fmt.Sprintf("table [%s] planned %d shard(s)", tableName, len(wheres)))
	}
	return wheres
}
