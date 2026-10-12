package application

import (
	"context"
	"testing"
	"time"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupBizHandleDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	require.NoError(t, db.AutoMigrate(&entity.Procinst{}))
	sqlDb, err := db.DB()
	require.NoError(t, err)
	sqlDb.SetMaxOpenConns(1)
	sqlDb.SetMaxIdleConns(1)

	previous := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = previous })
	return db
}

// seedProcinst 落一条流程实例并返回其主键
func seedProcinst(t *testing.T, repo base.Repo[*entity.Procinst], status entity.ProcinstStatus, bizStatus entity.ProcinstBizStatus) uint64 {
	t.Helper()
	procinst := &entity.Procinst{BizKey: "bk-1", Status: status, BizStatus: bizStatus}
	require.NoError(t, repo.Insert(context.Background(), procinst))
	return procinst.Id
}

func bizStatusOf(t *testing.T, db *gorm.DB, procinstId uint64) entity.ProcinstBizStatus {
	t.Helper()
	var stored entity.Procinst
	require.NoError(t, db.First(&stored, procinstId).Error)
	return stored.BizStatus
}

// TestClaimBizHandleSingleWinner 领取必须是排他的：快速路径与清扫任务撞上同一行时，
// 只能有一方判赢。两方都赢等于同一条 SQL 在目标库上执行两次
func TestClaimBizHandleSingleWinner(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &base.RepoImpl[*entity.Procinst]{}
	ctx := context.Background()
	id := seedProcinst(t, repo, entity.ProcinstStatusCompleted, entity.ProcinstBizStatusWait)

	first, err := claimBizHandle(ctx, repo, id)
	require.NoError(t, err)
	require.True(t, first, "the first claim must win")

	second, err := claimBizHandle(ctx, repo, id)
	require.NoError(t, err)
	require.False(t, second, "a row already taken out of Wait must not be claimable again")

	require.Equal(t, entity.ProcinstBizStatusDoing, bizStatusOf(t, db, id))
}

// TestClaimBizHandleIgnoresOtherRows 领取只影响目标行，别的待处理行必须还能被领
func TestClaimBizHandleIgnoresOtherRows(t *testing.T) {
	setupBizHandleDb(t)
	repo := &base.RepoImpl[*entity.Procinst]{}
	ctx := context.Background()

	first := seedProcinst(t, repo, entity.ProcinstStatusCompleted, entity.ProcinstBizStatusWait)
	second := seedProcinst(t, repo, entity.ProcinstStatusCompleted, entity.ProcinstBizStatusWait)

	won, err := claimBizHandle(ctx, repo, first)
	require.NoError(t, err)
	require.True(t, won)

	won, err = claimBizHandle(ctx, repo, second)
	require.NoError(t, err)
	require.True(t, won, "claiming one row must not consume another row's claim")
}

// TestSweepRedrivesWaitRows 兜住「提交成功、执行前崩溃」的窗口：仍停在待处理的行要被重驱动
func TestSweepRedrivesWaitRows(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &base.RepoImpl[*entity.Procinst]{}
	ctx := context.Background()

	id := seedProcinst(t, repo, entity.ProcinstStatusCompleted, entity.ProcinstBizStatusWait)

	var replayed []uint64
	count, err := sweepPendingBizHandles(ctx, repo, 10, func(_ context.Context, procinstId uint64) {
		replayed = append(replayed, procinstId)
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, []uint64{id}, replayed)
	require.Equal(t, entity.ProcinstBizStatusDoing, bizStatusOf(t, db, id))

	// 第二轮没有待处理行可捞，绝不重复回放
	count, err = sweepPendingBizHandles(ctx, repo, 10, func(context.Context, uint64) {
		t.Fatal("an already claimed row must not be replayed again")
	})
	require.NoError(t, err)
	require.Zero(t, count)
}

// TestSweepSkipsDoingRows 执行中的行不自动重来：回放内容是对目标库执行 SQL / 下发命令，
// 不保证幂等，盲目重跑会把一次已生效的变更再执行一遍。这类行靠告警暴露、由人判断
func TestSweepSkipsDoingRows(t *testing.T) {
	setupBizHandleDb(t)
	repo := &base.RepoImpl[*entity.Procinst]{}
	ctx := context.Background()

	seedProcinst(t, repo, entity.ProcinstStatusCompleted, entity.ProcinstBizStatusDoing)

	count, err := sweepPendingBizHandles(ctx, repo, 10, func(context.Context, uint64) {
		t.Fatal("a row currently being handled must never be re-driven automatically")
	})
	require.NoError(t, err)
	require.Zero(t, count)
}

// TestSweepIgnoresUnfinishedProcinst 未完成的流程实例即使业务状态是待处理也不能回放：
// 工单还在审批中就把 SQL 打到生产库，是最严重的越权执行
func TestSweepIgnoresUnfinishedProcinst(t *testing.T) {
	setupBizHandleDb(t)
	repo := &base.RepoImpl[*entity.Procinst]{}
	ctx := context.Background()

	for _, status := range []entity.ProcinstStatus{
		entity.ProcinstStatusActive, entity.ProcinstStatusCancelled, entity.ProcinstStatusTerminated,
	} {
		seedProcinst(t, repo, status, entity.ProcinstBizStatusWait)
	}

	count, err := sweepPendingBizHandles(ctx, repo, 10, func(context.Context, uint64) {
		t.Fatal("biz handle must not run for a process instance that is not completed")
	})
	require.NoError(t, err)
	require.Zero(t, count)
}

// TestSweepRespectsBatchLimit 单轮批量必须受限，避免积压时一轮打爆目标库
func TestSweepRespectsBatchLimit(t *testing.T) {
	setupBizHandleDb(t)
	repo := &base.RepoImpl[*entity.Procinst]{}
	ctx := context.Background()

	for range 5 {
		seedProcinst(t, repo, entity.ProcinstStatusCompleted, entity.ProcinstBizStatusWait)
	}

	var seen int
	count, err := sweepPendingBizHandles(ctx, repo, 2, func(context.Context, uint64) { seen++ })
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Equal(t, 2, seen)

	// 剩余待处理行留在原状态，下一轮继续
	remaining, err := repo.CountByCond(model.NewCond().EqStrict("biz_status", entity.ProcinstBizStatusWait))
	require.NoError(t, err)
	require.Equal(t, int64(3), remaining)
}

// sweepProcinstRepo 用真实仓储实现 domain 接口，让测试能构造出可运行的 procinstAppImpl
type sweepProcinstRepo struct {
	base.RepoImpl[*entity.Procinst]
}

func (r *sweepProcinstRepo) GetPageList(*entity.ProcinstQuery, ...string) (*model.PageResult[*entity.Procinst], error) {
	panic("not used by the biz handle path")
}

// TestSweepSkipsFuturePlannedRows 定时执行的核心约束：计划执行时间未到点的行不能被清扫领取，
// 必须留在 Wait，否则就提前执行了（审批通过不等于到点执行）
func TestSweepSkipsFuturePlannedRows(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &base.RepoImpl[*entity.Procinst]{}
	ctx := context.Background()
	future := time.Now().Add(time.Hour)
	p := &entity.Procinst{
		BizKey: "bk-future", Status: entity.ProcinstStatusCompleted, BizStatus: entity.ProcinstBizStatusWait, PlannedExecTime: &future,
	}
	require.NoError(t, repo.Insert(ctx, p))

	count, err := sweepPendingBizHandles(ctx, repo, 10, func(context.Context, uint64) {
		t.Fatal("a future-scheduled row must not be claimed")
	})
	require.NoError(t, err)
	require.Zero(t, count)
	require.Equal(t, entity.ProcinstBizStatusWait, bizStatusOf(t, db, p.Id), "未到点的工单必须仍停在待处理")
}

// TestSweepClaimsDuePlannedRows 计划时间已到点的行才能被清扫领取
func TestSweepClaimsDuePlannedRows(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &base.RepoImpl[*entity.Procinst]{}
	ctx := context.Background()
	past := time.Now().Add(-time.Minute)
	p := &entity.Procinst{
		BizKey: "bk-due", Status: entity.ProcinstStatusCompleted, BizStatus: entity.ProcinstBizStatusWait, PlannedExecTime: &past,
	}
	require.NoError(t, repo.Insert(ctx, p))

	count, err := sweepPendingBizHandles(ctx, repo, 10, func(_ context.Context, id uint64) {
		require.Equal(t, p.Id, id)
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, entity.ProcinstBizStatusDoing, bizStatusOf(t, db, p.Id))
}

// newAdminCtx 以管理员身份注入登录态，让改期/取消的属主校验直接通过（用例聚焦状态机与 CAS）
func newAdminCtx() context.Context {
	return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: consts.AdminId})
}

func newSchedApp(repo *sweepProcinstRepo) *procinstAppImpl {
	return &procinstAppImpl{AppImpl: base.AppImpl[*entity.Procinst, repository.Procinst]{Repo: repo}}
}

func seedCompletedWait(t *testing.T, repo base.Repo[*entity.Procinst]) uint64 {
	t.Helper()
	p := &entity.Procinst{BizKey: "bk-sched", Status: entity.ProcinstStatusCompleted, BizStatus: entity.ProcinstBizStatusWait}
	require.NoError(t, repo.Insert(context.Background(), p))
	return p.Id
}

// TestRescheduleBizHandleUpdatesPending 已批准未执行的工单可改期：写入计划时间且仍停在待处理
func TestRescheduleBizHandleUpdatesPending(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &sweepProcinstRepo{}
	id := seedCompletedWait(t, repo)
	app := newSchedApp(repo)

	at := time.Now().Add(2 * time.Hour)
	require.NoError(t, app.RescheduleBizHandle(newAdminCtx(), id, &at))

	require.Equal(t, entity.ProcinstBizStatusWait, bizStatusOf(t, db, id), "改期后仍应是待处理")
	var stored entity.Procinst
	require.NoError(t, db.First(&stored, id).Error)
	require.NotNil(t, stored.PlannedExecTime)
	require.True(t, stored.PlannedExecTime.After(time.Now().Add(time.Hour)), "计划时间应写入为所选的未来时间")
}

// TestCancelBizHandleSetsNo 取消未执行的回放：业务置为不处理（不改流程状态），并清空计划时间
func TestCancelBizHandleSetsNo(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &sweepProcinstRepo{}
	future := time.Now().Add(time.Hour)
	p := &entity.Procinst{
		BizKey: "bk-cancel", Status: entity.ProcinstStatusCompleted, BizStatus: entity.ProcinstBizStatusWait, PlannedExecTime: &future,
	}
	require.NoError(t, repo.Insert(context.Background(), p))
	app := newSchedApp(repo)

	require.NoError(t, app.CancelBizHandle(newAdminCtx(), p.Id))
	require.Equal(t, entity.ProcinstBizStatusNo, bizStatusOf(t, db, p.Id), "取消执行应把业务置为不处理")
	var stored entity.Procinst
	require.NoError(t, db.First(&stored, p.Id).Error)
	require.Nil(t, stored.PlannedExecTime, "取消执行必须同时清空计划时间，不留未来的歧义状态")
}

// TestRescheduleRejectsNonPendingRow 非「待处理」的工单（已在执行中）必须拒绝改期，不能改写正在跑的回放
func TestRescheduleRejectsNonPendingRow(t *testing.T) {
	setupBizHandleDb(t)
	repo := &sweepProcinstRepo{}
	p := &entity.Procinst{BizKey: "bk-doing", Status: entity.ProcinstStatusCompleted, BizStatus: entity.ProcinstBizStatusDoing}
	require.NoError(t, repo.Insert(context.Background(), p))
	app := newSchedApp(repo)

	at := time.Now().Add(time.Hour)
	require.Error(t, app.RescheduleBizHandle(newAdminCtx(), p.Id, &at), "非待处理态必须拒绝改期")
}

// TestRescheduleToPastTriggersExecution 改到过去/立即时，应同步触发一次回放而非等下一轮清扫；
// 未注册业务类型会记为失败终态，关键是它离开了 Wait（去不掉触发行则仍停 Wait 而变红）
func TestRescheduleToPastTriggersExecution(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &sweepProcinstRepo{}
	p := &entity.Procinst{
		BizKey: "bk-run-now", BizType: "unregistered-biz-type",
		Status: entity.ProcinstStatusCompleted, BizStatus: entity.ProcinstBizStatusWait,
	}
	require.NoError(t, repo.Insert(context.Background(), p))
	app := newSchedApp(repo)

	past := time.Now().Add(-time.Minute)
	require.NoError(t, app.RescheduleBizHandle(newAdminCtx(), p.Id, &past))
	require.NotEqual(t, entity.ProcinstBizStatusWait, bizStatusOf(t, db, p.Id), "改到已到期必须立即触发回放，而非停在待处理")
}

// TestSweepActuallyReachesTerminalState 清扫必须真的把业务跑完并落到终态。
//
// 这条用例钉的是**接线**而不是算法：清扫内部已经领取过（Wait→Doing），若它把「执行」写成
// 带领取的 ExecuteBizHandle，第二次领取必然判赢失败而什么都不做，行就永久停在 Doing——
// 日志与计数看起来都正常。注入假回调的单元测试测不出这种错误，只有走真实 app 才能暴露
func TestSweepActuallyReachesTerminalState(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &sweepProcinstRepo{}
	require.NoError(t, repo.Insert(context.Background(), &entity.Procinst{
		BizKey: "bk-sweep", BizType: "unregistered-biz-type", Status: entity.ProcinstStatusCompleted,
		BizStatus: entity.ProcinstBizStatusWait,
	}))

	app := &procinstAppImpl{
		AppImpl: base.AppImpl[*entity.Procinst, repository.Procinst]{Repo: repo},
	}

	count, err := app.SweepPendingBizHandles(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count, "sweep must report the row it re-drove")

	var stored entity.Procinst
	require.NoError(t, db.First(&stored).Error)
	if stored.BizStatus == entity.ProcinstBizStatusDoing {
		t.Fatal("biz status stuck in Doing: the sweep claimed the row but never executed it")
	}
	if stored.BizStatus != entity.ProcinstBizStatusFail {
		t.Errorf("biz status = %d, want Fail（未注册的业务处理器必须以失败记账）", stored.BizStatus)
	}
	require.NotEmpty(t, stored.BizHandleRes, "失败原因必须落库，否则只能看出没成功而不知道为何")
}

// TestCancelProcRejectsNonCancellableStatuses 取消只能作用于「审批中/挂起/退回」：
// 已完成实例上的「已批准、待执行」工单、已终止/已取消的实例都必须被拒绝且原状保留。
//
// 无条件保存的旧实现会把已完成（甚至已执行完）的实例改写成「已取消」——审计上留下假记录，
// 「已批准、待执行」的工单也会被从清扫链路里静默吞掉（那是 CancelBizHandle 的管辖范围）
func TestCancelProcRejectsNonCancellableStatuses(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &sweepProcinstRepo{}
	app := newSchedApp(repo)
	ctx := newAdminCtx()

	for _, status := range []entity.ProcinstStatus{
		entity.ProcinstStatusCompleted, entity.ProcinstStatusTerminated, entity.ProcinstStatusCancelled,
	} {
		p := &entity.Procinst{BizKey: "bk-reject", Status: status, BizStatus: entity.ProcinstBizStatusWait}
		require.NoError(t, repo.Insert(context.Background(), p))

		require.Error(t, app.CancelProc(ctx, p.Id), "status %d must not be cancellable", status)

		var stored entity.Procinst
		require.NoError(t, db.First(&stored, p.Id).Error)
		require.Equal(t, status, stored.Status, "被拒绝的取消绝不能改写流程状态")
		require.Equal(t, entity.ProcinstBizStatusWait, stored.BizStatus, "被拒绝的取消绝不能改写业务状态")
	}
}

// TestCancelProcCancelsPendingStatuses 审批中/挂起/退回三种状态都可被属主取消：
// 状态置「已取消」、业务置「不处理」（未回放的工单不再进入清扫）、并落下结束时间
func TestCancelProcCancelsPendingStatuses(t *testing.T) {
	db := setupBizHandleDb(t)
	repo := &sweepProcinstRepo{}
	app := newSchedApp(repo)
	ctx := newAdminCtx()

	for _, status := range []entity.ProcinstStatus{
		entity.ProcinstStatusActive, entity.ProcinstStatusSuspended, entity.ProcinstStatusBack,
	} {
		p := &entity.Procinst{BizKey: "bk-cancel", Status: status, BizStatus: entity.ProcinstBizStatusWait}
		require.NoError(t, repo.Insert(context.Background(), p))

		require.NoError(t, app.CancelProc(ctx, p.Id))

		var stored entity.Procinst
		require.NoError(t, db.First(&stored, p.Id).Error)
		require.Equal(t, entity.ProcinstStatusCancelled, stored.Status)
		require.Equal(t, entity.ProcinstBizStatusNo, stored.BizStatus)
		require.NotNil(t, stored.EndTime, "取消必须落下结束时间")
	}
}
