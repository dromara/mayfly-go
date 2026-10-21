package repository

import (
	"context"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

// RunAggregate 运行聚合统计（治理看板用；窗口过滤由实现决定）。
// AvgCostMs 用 float64 承接 SQL AVG（sqlite/mysql 驱动返回形态不同，int 扫描易翻车）
//
// 刻意不再携带 succeeded/failed/canceled 三个固定字段：那等于在聚合结果里再抄一份终态清单，
// 新增终态时它静默缺席（看板分布缺格、成功率分母虚小）。状态分布改走 CountGroupedByStatus。
type RunAggregate struct {
	Total            int64   `gorm:"column:total"`
	PromptTokens     int64   `gorm:"column:prompt_tokens"`
	CompletionTokens int64   `gorm:"column:completion_tokens"`
	TotalTokens      int64   `gorm:"column:total_tokens"`
	AvgCostMs        float64 `gorm:"column:avg_cost_ms"`
}

// Run 运行聚合仓储
type Run interface {
	base.Repo[*entity.Run]

	// GetPageList 分页查询运行记录（按触发类型/状态/装配档/触发源/归属账号过滤）
	GetPageList(condition *entity.Run, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Run], error)

	// GetByIdempotencyKey 按幂等键查询已存在的运行（去重短路用）；不存在返回 model.ErrNotFound
	GetByIdempotencyKey(key string) (*entity.Run, error)

	// ExistsActiveByTriggerRef 判断指定触发源是否存在「非终态」运行（entity.ActiveRunStatuses）。
	// 供 cron 触发的重叠防护：同一任务上一次运行尚未收尾时跳过本次触发，避免异步模型下堆叠。
	ExistsActiveByTriggerRef(ctx context.Context, triggerType, triggerRef string) (bool, error)

	// ClaimNextQueued 原子认领最早入队的运行（queued→running）并写入在途归属实例，多实例/多 worker 安全。
	//
	// 认领靠「WHERE status=queued 的条件更新」裁决：并发下只有一个调用能改到行。
	// 无可认领（队列为空或被抢先）时返回 (nil, nil)，调用方据此结束本轮 drain。
	// runnerInstance 与状态流转同一条 UPDATE 写入（认领与归属必须原子，见实现注释）。
	ClaimNextQueued(ctx context.Context, startedAt time.Time, runnerInstance string) (*entity.Run, error)

	// Aggregate 窗口内（createTime >= since；since 零值=全量）运行聚合统计。
	// 标准 SQL 聚合（SUM/AVG/CASE WHEN），跨 sqlite/mysql/pg 方言可移植
	Aggregate(ctx context.Context, since time.Time) (*RunAggregate, error)

	// CountGroupedByStatus 窗口内按状态分组计数；since 零值=全量。
	// 看板的「状态分布」与「成功率分母」都取它，不再在 SQL / 应用层各自拄一份状态清单
	CountGroupedByStatus(ctx context.Context, since time.Time) (map[string]int64, error)
}
