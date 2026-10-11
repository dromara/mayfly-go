package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/taskx"
	"time"
)

const (
	// bizHandleSweepKey 业务回放清扫任务的分布式锁键：多实例部署时只有一个实例执行清扫
	bizHandleSweepKey = "flow-biz-handle-sweep"
	// bizHandleSweepInterval 清扫间隔。审批通过的正常路径由提交后立即执行覆盖，清扫只兜崩溃遗留
	bizHandleSweepInterval = "@every 1m"
	// bizHandleSweepBatchSize 单轮最多重驱动条数。截断发生在取回之后（读入口不收 ctx，也就
	// 无法安全地加 LIMIT：LIMIT 进了条件对象，CountByCond 会跟着一起被限制）；
	// 待处理行只来自「提交后崩溃」这一窗口，正常量级很小，若真出现大批积压需要给读入口补 ctx 分页
	bizHandleSweepBatchSize = 50
	// bizHandleDoingWarnAfter 处于「执行中」超过该时长即告警：说明执行者中途崩溃或被卡住
	bizHandleDoingWarnAfter = 30 * time.Minute
	// bizHandleSweepLockTTL 清扫任务的分布式锁 TTL，与「执行中」告警阈值解耦：持锁实例崩溃后，
	// 到点的定时工单最多被阻塞该时长即可被其他实例接管（单行领取另有 CAS 判赢，锁只用于少起几轮空扫，
	// 取分钟级即可，不必等于 30min）
	bizHandleSweepLockTTL = 10 * time.Minute
)

// claimBizHandle 把某条流程实例的业务回放从「待处理」原子占到「执行中」，按影响行数判赢。
//
// 快速路径（审批请求提交后立即执行）与清扫任务可能在同一行上撞上，这里就是它们的唯一裁决点：
// 判赢失败的一方必须什么都不做，否则同一条 SQL 会在目标实例上执行两次。
//
// 到期谓词与清扫取数条件一致（计划时间为空或已到点）：裁决点自洽，避免「取数后、领取前被改期到未来」
// 的窗口里把一条已改到未来的工单提前执行。repo 由参数传入而非内部取 IOC 单例，是为了让这段并发裁决能被直接单测覆盖
func claimBizHandle(ctx context.Context, repo base.Repo[*entity.Procinst], procinstId uint64) (bool, error) {
	rows, err := repo.UpdateColsByCond(ctx, base.NewPatch().Set("biz_status", entity.ProcinstBizStatusDoing),
		model.NewCond().Eq(model.IdColumn, procinstId).EqStrict("biz_status", entity.ProcinstBizStatusWait).
			And("(planned_exec_time IS NULL OR planned_exec_time <= ?)", time.Now()))
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

// sweepPendingBizHandles 重驱动崩溃遗留的「已完成但业务待处理」实例。
//
// 只捞 Wait：处于 Doing 的行**不会被自动重来**。回放的内容是对目标库执行 SQL / 对机器下发命令，
// 这类动作不保证幂等，盲目重跑可能把一次已生效的变更再执行一遍。因此执行者中途崩溃时，
// 该行停在 Doing 并被告警出来，由人确认目标侧现状后再决定是否重投
func sweepPendingBizHandles(ctx context.Context, repo base.Repo[*entity.Procinst], limit int, run func(ctx context.Context, procinstId uint64)) (int, error) {
	if limit <= 0 {
		limit = bizHandleSweepBatchSize
	}
	pending, err := repo.SelectByCond(model.NewCond().
		EqStrict("status", entity.ProcinstStatusCompleted).
		EqStrict("biz_status", entity.ProcinstBizStatusWait).
		// 只捞到期的：未设计划时间=立即可领；设了计划时间则必须已到点，未到点留待下一轮清扫
		And("(planned_exec_time IS NULL OR planned_exec_time <= ?)", time.Now()), "id")
	if err != nil {
		return 0, err
	}
	// 见 bizHandleSweepBatchSize 的说明：这里按内存截断，不做 SQL LIMIT
	if len(pending) > limit {
		pending = pending[:limit]
	}

	swept := 0
	for _, procinst := range pending {
		won, err := claimBizHandle(ctx, repo, procinst.Id)
		if err != nil {
			logx.ErrorfContext(ctx, "failed to claim biz handle for procinst %d: %v", procinst.Id, err)
			continue
		}
		if !won {
			// 快速路径已抢先领取，属正常态
			continue
		}
		run(ctx, procinst.Id)
		swept++
	}
	return swept, nil
}

// warnStuckBizHandles 告警长期停在「执行中」的回放，它们是崩溃或被卡住的唯一可见痕迹
func warnStuckBizHandles(ctx context.Context, repo base.Repo[*entity.Procinst]) {
	stuck, err := repo.CountByCond(model.NewCond().
		EqStrict("biz_status", entity.ProcinstBizStatusDoing).
		And("update_time < ?", time.Now().Add(-bizHandleDoingWarnAfter)))
	if err != nil {
		logx.ErrorfContext(ctx, "failed to count stuck biz handles: %v", err)
		return
	}
	if stuck > 0 {
		logx.WarnfContext(ctx, "%d process instance(s) stuck in handling biz for over %v: the executor may have crashed, check the target side then re-trigger", stuck, bizHandleDoingWarnAfter)
	}
}

// ExecuteBizHandle 领取并执行某条流程实例的业务回放。
//
// 未抢到领取权时直接返回：说明已有执行者在跑，本调用什么都不做
func (p *procinstAppImpl) ExecuteBizHandle(ctx context.Context, procinstId uint64) error {
	won, err := claimBizHandle(ctx, p.GetRepo(), procinstId)
	if err != nil {
		return err
	}
	if !won {
		return nil
	}
	return p.runClaimedBizHandle(ctx, procinstId)
}

// runClaimedBizHandle 执行**已被本次调用领取**的业务回放。
//
// 领取与执行必须分成两个可各自取用的函数：清扫任务先领取再执行，若把「执行」写成
// ExecuteBizHandle，它内部的第二次领取会因状态已不是待处理而判赢失败，于是清扫只做了一件
// 坏事——把行从待处理改成执行中，却一条业务都没执行（看起来仍在正常工作，日志还打印重驱动条数）
func (p *procinstAppImpl) runClaimedBizHandle(ctx context.Context, procinstId uint64) error {
	procinst, err := p.GetById(procinstId)
	if err != nil {
		// 领取已生效但没执行成，状态会停在 Doing 并被清扫任务告警出来，不能在此悄悄复位
		return err
	}
	p.bizHandle(ctx, procinst)
	return nil
}

// SweepPendingBizHandles 供清扫任务与运维入口调用的一轮重驱动
func (p *procinstAppImpl) SweepPendingBizHandles(ctx context.Context) (int, error) {
	repo := p.GetRepo()
	warnStuckBizHandles(ctx, repo)
	return sweepPendingBizHandles(ctx, repo, bizHandleSweepBatchSize, func(runCtx context.Context, procinstId uint64) {
		// 这里的行是本次清扫刚刚领取到的，必须直接执行，不能再走带领取的 ExecuteBizHandle
		if err := p.runClaimedBizHandle(runCtx, procinstId); err != nil {
			logx.ErrorfContext(runCtx, "failed to replay biz handle for procinst %d: %v", procinstId, err)
		}
	})
}

// InitBizHandleSweep 绑定业务回放清扫任务。
//
// 审批通过的正常路径靠提交后立即执行，这条任务只负责兜住「提交成功、执行前进程退出」的窗口，
// 因此崩溃恢复后那条 SQL 仍会被执行，而不是无痕丢失
func InitBizHandleSweep() {
	app := GetProcinstApp()
	if err := taskx.BindCronTaskWithLock(bizHandleSweepKey, bizHandleSweepInterval, bizHandleSweepLockTTL, true, func() {
		// 定时任务没有请求上下文：造一个带 traceId 的任务级 ctx，这一轮的领取与回写才能与日志串联
		taskCtx := contextx.WithTraceId(context.Background())
		defer gox.Recover()
		count, err := app.SweepPendingBizHandles(taskCtx)
		if err != nil {
			logx.Errorf("failed to sweep pending biz handles: %v", err)
			return
		}
		if count > 0 {
			logx.Infof("replayed %d pending biz handle(s)", count)
		}
	}); err != nil {
		logx.Errorf("failed to bind biz handle sweep task: %v", err)
	}
}

// runBizHandleAfterCommit 在事务提交成功后立即尝试执行本次审批的业务回放。
//
// 只保证「不提前做」：进程若在提交后、执行前退出，这一条会被清扫任务按 Wait 状态重新驱动。
// 这里返回的错误不阻断请求（记账已完成），执行失败会留在 Doing/Wait 状态上由告警暴露
//
// 计划执行时间在未来时不立即回放：留在 Wait，由清扫任务按到期时间领取（定时执行）。
func runBizHandleAfterCommit(ctx context.Context, procinst *entity.Procinst) error {
	if procinst.PlannedExecTime != nil && procinst.PlannedExecTime.After(time.Now()) {
		return nil
	}
	app := GetProcinstApp()
	if err := app.ExecuteBizHandle(ctx, procinst.Id); err != nil {
		return fmt.Errorf("failed to replay biz handle for procinst %d: %w", procinst.Id, err)
	}
	return nil
}
