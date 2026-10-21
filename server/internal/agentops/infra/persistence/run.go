package persistence

import (
	"context"
	"strings"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type runRepoImpl struct {
	base.RepoImpl[*entity.Run]
}

var _ repository.Run = (*runRepoImpl)(nil)

func newRunRepo() repository.Run {
	return &runRepoImpl{}
}

// GetPageList 分页查询运行记录（列名 camelCase 由基座统一蛇形化）。
//
// 过滤维度与 entity.Run 上带 form tag 的字段一一对应：form tag 等于对外声明「本接口可按此筛」，
// 声明了却不拼谓词就是静默忽略——调用方带上 parentRunId 请求编排子运行，拿回的是全量运行，
// 而接口照返 200，看起来像「这次编排没有子运行」。新增 form tag 时必须同步在此补一行
func (r *runRepoImpl) GetPageList(condition *entity.Run, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Run], error) {
	qd := model.NewCond().
		Eq("triggerType", condition.TriggerType).
		Eq("triggerRef", condition.TriggerRef).
		Eq("status", condition.Status).
		Eq("profileId", condition.ProfileId).
		Eq("ownerAccountId", condition.OwnerAccountId).
		Eq("conversationId", condition.ConversationId).
		Eq("parentRunId", condition.ParentRunId).
		// Feedback 只能筛 up/down：Eq 跳零值的基座语义使「未反馈」无法用空串表达，
		// 列表侧靠列本身渲「未反馈」标记即可看出，不为此造哨兵值（哨兵会和真实取值同列存储）
		Eq("feedback", condition.Feedback).
		OrderBy(orderBy...)
	return r.PageByCond(qd, pageParam)
}

// GetByIdempotencyKey 按幂等键精确查询（幂等键恒非空，用 EqStrict 避免零值跳过语义干扰）
func (r *runRepoImpl) GetByIdempotencyKey(key string) (*entity.Run, error) {
	run := new(entity.Run)
	if err := r.GetByCond(model.NewCond().EqStrict("idempotencyKey", key).Dest(run)); err != nil {
		return nil, err
	}
	return run, nil
}

// ClaimNextQueued 原子认领最早入队的运行，并把在途归属记到本实例。
//
// 两步：① 取最早 queued（显式按主键升序 + LIMIT 1，即 FIFO）；② 条件更新 queued→running，
// 仅当仍为 queued 才改到行——并发竞争由「改动行数==1」裁决，落败者返回 (nil, nil) 让调用方重试。
// 归属列与状态流转同一条 UPDATE 写入：认领成功即归属确定，避免出现「已 running 但归属为空」
// 的中间行（注：残留清扫按租约不按归属列，见 RunService.SweepInterruptedRuns）。
func (r *runRepoImpl) ClaimNextQueued(ctx context.Context, startedAt time.Time, runnerInstance string) (*entity.Run, error) {
	candidate := new(entity.Run)
	// 排序必须显式写定「按主键升序取最早」：不写时拿到的并不是最早一条（实测同秒入队时
	// 会先拿到后一条），慢队列里早入队的运行会被持续后来的挤后，表现为「排在前面永远轮不到」
	if err := r.GetByCond(model.NewCond().EqStrict("status", entity.RunStatusQueued).OrderByAsc("id").Dest(candidate)); err != nil {
		if model.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	rows, err := r.UpdateColsByCond(ctx,
		base.NewPatch().
			Set("status", entity.RunStatusRunning).
			Set("startedAt", startedAt).
			Set("runnerInstance", runnerInstance),
		model.NewCond().EqStrict("id", candidate.Id).EqStrict("status", entity.RunStatusQueued))
	if err != nil {
		return nil, err
	}
	if rows != 1 {
		// 被其它 worker/实例抢先认领
		return nil, nil
	}

	// 重新载入完整运行（含刚写入的 running/startedAt，供执行期使用）
	claimed, err := r.GetById(candidate.Id)
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// ExistsActiveByTriggerRef 判断触发源是否存在非终态运行（重叠防护）。
// 非终态集合真源为 entity.ActiveRunStatuses；triggerType/triggerRef 恒非空用 EqStrict 精确匹配。
func (r *runRepoImpl) ExistsActiveByTriggerRef(ctx context.Context, triggerType, triggerRef string) (bool, error) {
	return r.Exists(ctx, model.NewCond().
		EqStrict("triggerType", triggerType).
		EqStrict("triggerRef", triggerRef).
		In("status", entity.ActiveRunStatuses))
}

// Aggregate 窗口内运行聚合统计。
// 状态字面量由实体常量拼接进 SQL（非常量用户输入，无注入面）；SUM/AVG/CASE WHEN 均为标准 SQL，
// 跨 sqlite/mysql/pg 可移植。Run 用 ModelNLD（无逻辑删除列），无需附加 is_deleted 过滤。
//
// 平均耗时只统计「已进入终态且真跑过」的运行（AVG 忽略 NULL）：进行中运行 cost_ms 恒为 0，
// 未启动就被取消的运行 cost_ms 也为 0，两者纳入都会稀释指标——急停清场几十条慢运行后
// 「平均耗时」会反而大幅下降，运维据此误判系统变快。终态集合取 entity.TerminalRunStatuses，
// 新增终态不需回来改这段 SQL（状态分布本身改走 CountGroupedByStatus）。
func (r *runRepoImpl) Aggregate(ctx context.Context, since time.Time) (*repository.RunAggregate, error) {
	terminalIn := "status IN ('" + strings.Join(entity.TerminalRunStatuses, "', '") + "')"
	// since 零值=全量（接口契约）：必须不拼这条谓词。恒拼则会踩两重跨方言不一致：
	// sqlite 下 TEXT 亲和比较直接 0 行，mysql 下绑定 0001-01-01 超出 DATETIME 合法范围（严格模式报错）
	where := ""
	args := make([]any, 0, 1)
	if !since.IsZero() {
		where = " WHERE create_time >= ?"
		args = append(args, since)
	}
	query := "SELECT COUNT(*) AS total, " +
		"COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(completion_tokens), 0) AS completion_tokens, " +
		"COALESCE(SUM(total_tokens), 0) AS total_tokens, " +
		"COALESCE(AVG(CASE WHEN " + terminalIn + " AND cost_ms > 0 THEN cost_ms END), 0) AS avg_cost_ms " +
		"FROM " + new(entity.Run).TableName() + where
	res := new(repository.RunAggregate)
	if err := r.SelectBySQL(ctx, query, res, args...); err != nil {
		return nil, err
	}
	return res, nil
}

// CountGroupedByStatus 按状态分组计数（一次查询覆盖全部状态，供治理看板分布与成功率分母用）。
// since 零值=全量，与 Aggregate 同一口径。
func (r *runRepoImpl) CountGroupedByStatus(ctx context.Context, since time.Time) (map[string]int64, error) {
	type row struct {
		Status string `gorm:"column:status"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	query := "SELECT status, COUNT(*) AS cnt FROM " + new(entity.Run).TableName()
	args := make([]any, 0, 1)
	if !since.IsZero() {
		query += " WHERE create_time >= ?"
		args = append(args, since)
	}
	query += " GROUP BY status"
	var rows []*row
	if err := r.SelectBySQL(ctx, query, &rows, args...); err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, it := range rows {
		out[it.Status] = it.Cnt
	}
	return out, nil
}
