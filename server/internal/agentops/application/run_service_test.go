package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/infra/persistence"
	"mayfly-go/internal/ai/memory"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/model"

	"github.com/cloudwego/eino/schema"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// runTestCtx 无登录账号的后台 ctx（服务身份由 Run.OwnerAccountId 承载，不依赖 ctx 账号）
func runTestCtx() context.Context {
	return context.Background()
}

var registerRepoOnce sync.Once

// setupRunSvc 打开 sqlite 内存库、建 Run 表、装配一个绑定真实仓储的 RunService。
//
// 用真实仓储 + 内存库而非 mock：Run 的核心价值就在「幂等唯一约束 / 原子认领 / 状态条件流转」
// 这些只有落到真实 SQL 语义上才测得准。headless.Runner 由用例注入 fake，隔离真实 LLM。
func setupRunSvc(t *testing.T, poolSize int) *runServiceAppImpl {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		NamingStrategy:       gormx.MetaNamingStrategy(),
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// :memory: 每连接独立库，锁死单连接避免建表与读写落在不同连接
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(&entity.Run{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	registerRepoOnce.Do(persistence.InitIoc)
	repo := ioc.Get[repository.Run]()
	require.NotNil(t, repo)

	svc := &runServiceAppImpl{PoolSize: poolSize}
	svc.Repo = repo
	svc.signalCh = make(chan struct{}, 1)
	svc.stopCh = make(chan struct{})
	svc.inflight = make(map[uint64]context.CancelFunc)
	t.Cleanup(svc.Stop)
	return svc
}

// fakeRunner 安装一个可编程的 headless.Runner，返回安装前的旧实现以便复原
func fakeRunner(t *testing.T, fn headless.Runner) {
	t.Helper()
	headless.Register(fn)
}

func enqueueOK(t *testing.T, svc *runServiceAppImpl, req *EnqueueReq) *entity.Run {
	t.Helper()
	run, err := svc.Enqueue(runTestCtx(), req)
	require.NoError(t, err)
	require.NotNil(t, run)
	return run
}

// waitStatus 轮询等待运行到达目标状态（避免固定 sleep 造成 flaky）
func waitStatus(t *testing.T, svc *runServiceAppImpl, runId uint64, want string) *entity.Run {
	t.Helper()
	var got *entity.Run
	require.Eventually(t, func() bool {
		r, err := svc.GetById(runId)
		if err != nil {
			return false
		}
		got = r
		return r.Status == want
	}, 5*time.Second, 20*time.Millisecond, "run %d did not reach status %s", runId, want)
	return got
}

func baseReq() *EnqueueReq {
	return &EnqueueReq{
		TriggerType:    entity.TriggerTypeManual,
		ProfileId:      headless.ProfileDiagnostician,
		Input:          "check db slow query",
		OwnerAccountId: 1,
		OwnerUsername:  "admin",
	}
}

// --- Enqueue 校验（fail-closed） ---

func TestEnqueueValidation(t *testing.T) {
	svc := setupRunSvc(t, 1)
	ctx := runTestCtx()

	_, err := svc.Enqueue(ctx, nil)
	require.Error(t, err, "nil 请求必须拒绝")

	noOwner := baseReq()
	noOwner.OwnerAccountId = 0
	_, err = svc.Enqueue(ctx, noOwner)
	require.Error(t, err, "缺运行身份必须 fail-closed 拒绝")

	empty := baseReq()
	empty.Input = ""
	empty.Instruction = ""
	_, err = svc.Enqueue(ctx, empty)
	require.Error(t, err, "既无 input 又无 instruction 必须拒绝")
}

// --- 幂等去重 ---

func TestEnqueueIdempotency(t *testing.T) {
	svc := setupRunSvc(t, 1)
	// fake 阻塞在成功返回：本用例只关注入队去重，不驱动执行
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusSucceeded}, nil
	})

	// 相同幂等键：第二次短路返回既有运行，库中只有一行
	r1 := baseReq()
	r1.IdempotencyKey = "alert:evt-100"
	run1 := enqueueOK(t, svc, r1)

	r2 := baseReq()
	r2.IdempotencyKey = "alert:evt-100"
	run2 := enqueueOK(t, svc, r2)
	require.Equal(t, run1.Id, run2.Id, "同幂等键必须命中既有运行")

	cnt, err := svc.CountByCond(&entity.Run{IdempotencyKey: "alert:evt-100"})
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt, "同幂等键只应落一行")

	// 无幂等键：自动生成天然唯一键，两次入队是两条独立运行
	a1 := enqueueOK(t, svc, baseReq())
	a2 := enqueueOK(t, svc, baseReq())
	require.NotEqual(t, a1.Id, a2.Id, "无幂等键的两次入队应各自独立")
	require.NotEqual(t, a1.IdempotencyKey, a2.IdempotencyKey)
}

// --- 工作池执行：成功路径 ---

func TestRunWorkerExecutesQueued(t *testing.T) {
	svc := setupRunSvc(t, 2)
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		require.Equal(t, headless.ProfileDiagnostician, req.ProfileId)
		require.True(t, req.Persist, "运行默认应持久化以获得审计/回放")
		require.Equal(t, uint64(1), req.OwnerAccountId, "owner 身份须透传给无头运行")
		return &headless.RunResult{
			Output:         "root cause: missing index",
			ConversationId: 9527,
			TurnId:         "turn-1",
			Status:         headless.RunStatusSucceeded,
			Usage:          tokUsage(11, 22, 33),
		}, nil
	})

	run := enqueueOK(t, svc, baseReq())
	svc.Start()

	got := waitStatus(t, svc, run.Id, entity.RunStatusSucceeded)
	require.Equal(t, "root cause: missing index", got.Summary)
	require.Equal(t, uint64(9527), got.ConversationId)
	require.Equal(t, "turn-1", got.TurnId)
	require.Equal(t, int64(11), got.PromptTokens)
	require.Equal(t, int64(22), got.CompletionTokens)
	require.Equal(t, int64(33), got.TotalTokens)
	require.NotNil(t, got.StartedAt)
	require.NotNil(t, got.FinishedAt)
	require.GreaterOrEqual(t, got.CostMs, int64(0))
	require.Empty(t, got.Err)
}

// --- 工作池执行：失败路径 ---

func TestRunWorkerFailure(t *testing.T) {
	svc := setupRunSvc(t, 1)
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusFailed}, errors.New("llm upstream 500")
	})

	run := enqueueOK(t, svc, baseReq())
	svc.Start()

	got := waitStatus(t, svc, run.Id, entity.RunStatusFailed)
	require.Contains(t, got.Err, "llm upstream 500")
}

// --- 重试：仅超时/连接类可重试，业务失败不重试 ---

func TestRunRetryOnTimeout(t *testing.T) {
	svc := setupRunSvc(t, 1)
	var calls int32
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			return &headless.RunResult{Status: headless.RunStatusTimeout}, nil // 首次超时
		}
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: "ok after retry"}, nil
	})

	req := baseReq()
	req.RetryTimes = 1
	run := enqueueOK(t, svc, req)
	svc.Start()

	got := waitStatus(t, svc, run.Id, entity.RunStatusSucceeded)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "超时应重试一次后成功")
	require.Equal(t, "ok after retry", got.Summary)
}

func TestRunNoRetryOnBusinessFailure(t *testing.T) {
	svc := setupRunSvc(t, 1)
	var calls int32
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		atomic.AddInt32(&calls, 1)
		return &headless.RunResult{Status: headless.RunStatusFailed}, errors.New("business failure")
	})

	req := baseReq()
	req.RetryTimes = 3
	run := enqueueOK(t, svc, req)
	svc.Start()

	waitStatus(t, svc, run.Id, entity.RunStatusFailed)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "业务失败（非超时/连接）不得重试")
}

// --- MaxStep / Title 透传给无头运行 ---

func TestRunMaxStepAndTitlePassthrough(t *testing.T) {
	svc := setupRunSvc(t, 1)
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		require.Equal(t, 9, req.MaxStep, "MaxStep 须透传给无头运行")
		require.Equal(t, "夜巡", req.Title, "Title 须透传用于会话标题")
		return &headless.RunResult{Status: headless.RunStatusSucceeded}, nil
	})

	req := baseReq()
	req.MaxStep = 9
	req.Title = "夜巡"
	run := enqueueOK(t, svc, req)
	svc.Start()

	waitStatus(t, svc, run.Id, entity.RunStatusSucceeded)
}

// --- 取消运行中的 Run（急停/人工取消） ---

func TestRunCancel(t *testing.T) {
	svc := setupRunSvc(t, 1)
	// fake 阻塞直到 ctx 取消，模拟长运行；取消后回报 canceled（与真实 RunHeadless 一致）
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		<-ctx.Done()
		return &headless.RunResult{Status: headless.RunStatusCanceled}, ctx.Err()
	})

	run := enqueueOK(t, svc, baseReq())
	svc.Start()

	// 等待进入 running（已被 worker 认领并在 fake 中阻塞）
	waitStatus(t, svc, run.Id, entity.RunStatusRunning)

	require.NoError(t, svc.Cancel(runTestCtx(), run.Id))
	got := waitStatus(t, svc, run.Id, entity.RunStatusCanceled)
	require.Equal(t, entity.RunStatusCanceled, got.Status)

	// 终态后再取消必须明确报错：旧写法返回 nil 使前端弹「已取消」而库里什么都没变，
	// 急停清场的 canceled 计数也会被没改到行的运行虚增
	require.ErrorIs(t, svc.Cancel(runTestCtx(), run.Id), ErrRunNotCancelable)

	// 已落库的取消原因不被覆写
	again, gerr := svc.GetById(run.Id)
	require.NoError(t, gerr)
	require.Equal(t, entity.RunStatusCanceled, again.Status)
}

// 取消尚未被认领的 queued 运行：本进程不会为它注册 cancel，必须回收取消标记，
// 否则长期运行 + 频繁取消下 cancelReq 只增不减（旧写法只在「没改到行」与 untrackInflight 两条路清理）
func TestCancelQueuedRunReleasesCancelMark(t *testing.T) {
	svc := setupRunSvc(t, 1) // 不 Start：运行保持 queued，永远不会 trackInflight

	run := enqueueOK(t, svc, baseReq())
	require.NoError(t, svc.Cancel(runTestCtx(), run.Id))

	svc.inflightMu.Lock()
	pending := len(svc.cancelReq)
	svc.inflightMu.Unlock()
	require.Zero(t, pending, "取消排队运行后不得残留取消标记（map 无界增长）")
}

// TestCancelWritesCostMs 取消已起跑的运行必须补写 costMs。
//
// 不写的后果：canceled 是终态，会进入治理看板「平均耗时」的分母，以 0 计入拉低均值；
// 而 finish 的条件更新要求非终态，取消后没有任何路径会再替它补这个值——点一次急停清场
// 取消几十条慢运行，看板就显示系统「变快」了（旧用例把被稀释的 500 当成了期望值）。
// 未起跑的 queued 运行 StartedAt 为空，保持 0（聚合侧按 cost_ms > 0 排除）。
func TestCancelWritesCostMs(t *testing.T) {
	svc := setupRunSvc(t, 1)
	ctx := runTestCtx()

	// 起跑中的运行：手工置 running + started_at（工作池未启动，不会被认领走）
	running := enqueueOK(t, svc, baseReq())
	started := time.Now().Add(-5 * time.Second)
	_, err := svc.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().Set("status", entity.RunStatusRunning).Set("startedAt", started),
		model.NewCond().EqStrict("id", running.Id))
	require.NoError(t, err)

	require.NoError(t, svc.Cancel(ctx, running.Id))
	got := waitStatus(t, svc, running.Id, entity.RunStatusCanceled)
	require.GreaterOrEqual(t, got.CostMs, int64(4000), "取消必须按已跑时长补写 costMs（≈ 5s）")

	// 排队中被取消：从未起跑，不应凭空造出一个耗时
	queued := enqueueOK(t, svc, baseReq())
	require.NoError(t, svc.Cancel(ctx, queued.Id))
	gotQueued := waitStatus(t, svc, queued.Id, entity.RunStatusCanceled)
	require.Zero(t, gotQueued.CostMs, "未起跑就被取消的运行 cost_ms 保持 0，由聚合侧排除")
}

// 一次 panic 不得永久少一个 worker：Start 只按固定数量拉起，旧写法 recover 收在 worker 函数
// 出口，panic 后该 goroutine 直接退出 → 池只减不增，最终 queued 再无人认领且无告警
func TestWorkerSurvivesPanic(t *testing.T) {
	svc := setupRunSvc(t, 1)
	var calls atomic.Int32
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		if calls.Add(1) == 1 {
			panic("tool panicked unexpectedly")
		}
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: "ok"}, nil
	})

	panicked := enqueueOK(t, svc, baseReq())
	next := enqueueOK(t, svc, baseReq())
	svc.Start()

	failed := waitStatus(t, svc, panicked.Id, entity.RunStatusFailed)
	require.Contains(t, failed.Err, "panicked", "panic 必须补写终态与原因，不留悬挂 running")

	// 关键：同一个 worker 继续服务下一条（旧写法这里会停在 queued 直到超时失败）
	waitStatus(t, svc, next.Id, entity.RunStatusSucceeded)
}

// finish 必须按影响行数判赢：运行已被 Cancel 收尾时，内存里算出的 succeeded 不是库中真值，
// 不得覆写也不得外发通知（通知不可回滚）
func TestFinishYieldsWhenTerminalWrittenElsewhere(t *testing.T) {
	svc := setupRunSvc(t, 1)

	run := enqueueOK(t, svc, baseReq())
	require.NoError(t, svc.Cancel(runTestCtx(), run.Id)) // queued → canceled，未被认领

	// 模拟 worker 跑完后才回写终态（与 Cancel 竞态的那个窗口）
	status, won := svc.finish(runTestCtx(), run, &headless.RunResult{
		Status: headless.RunStatusSucceeded, Output: "late result",
	}, nil)
	require.Equal(t, entity.RunStatusSucceeded, status, "计算出的终态仍回传供日志使用")
	require.False(t, won, "没抢到终态写入时必须报告未抢到")

	got, err := svc.GetById(run.Id)
	require.NoError(t, err)
	require.Equal(t, entity.RunStatusCanceled, got.Status, "已取消的运行不被迟到的终态覆写")
	require.Empty(t, got.Summary, "被取消的运行不得落入晚到的结果内容")
}

// 租约清扫的两个方向：过期残留必须收尾（含其他实例/上一个进程留下的行），未过期（可能真有
// 实例在跑）不得误杀。不按 runner_instance 归属判定——实例 id 每进程随机，按归属清扫会让
// 上一个进程留下的 running 永久无人收敛，进而把同一任务的后续派发永久判为「还在跑」而跳过
func TestSweepInterruptedRunsByLeaseExpiry(t *testing.T) {
	svc := setupRunSvc(t, 1)

	mine := enqueueOK(t, svc, baseReq())   // 本实例认领后被强杀的残留
	theirs := enqueueOK(t, svc, baseReq()) // 其他实例崩溃留下的残留：旧实现按归属判定永远扫不到
	live := enqueueOK(t, svc, baseReq())   // 其他实例此刻正在跑

	now := time.Now()
	claimed, err := svc.GetRepo().ClaimNextQueued(runTestCtx(), now, runInstanceId)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.Equal(t, mine.Id, claimed.Id)
	require.Equal(t, runInstanceId, claimed.RunnerInstance, "认领时必须写入归属实例")
	for i := 0; i < 2; i++ {
		c, err := svc.GetRepo().ClaimNextQueued(runTestCtx(), now, "instance-other")
		require.NoError(t, err)
		require.NotNil(t, c)
	}

	// 把前两条的租约起点推到「默认超时 + 宽限期」之外，模拟执行者已消失
	expired := now.Add(-(defaultRunTimeout + time.Duration(runSweepGraceSeconds)*time.Second + time.Minute))
	for _, run := range []*entity.Run{mine, theirs} {
		_, err = svc.GetRepo().UpdateColsByCond(runTestCtx(), base.NewPatch().Set("startedAt", expired),
			model.NewCond().EqStrict("id", run.Id))
		require.NoError(t, err)
	}

	require.NoError(t, svc.SweepInterruptedRuns(runTestCtx()))

	gotMine, err := svc.GetById(mine.Id)
	require.NoError(t, err)
	require.Equal(t, entity.RunStatusFailed, gotMine.Status, "本实例过期残留必须被收尾")
	require.Contains(t, gotMine.Err, "lease expired")

	gotTheirs, err := svc.GetById(theirs.Id)
	require.NoError(t, err)
	require.Equal(t, entity.RunStatusFailed, gotTheirs.Status, "其他实例的过期残留同样要收尾（按归属判定会永久漏掉）")

	gotLive, err := svc.GetById(live.Id)
	require.NoError(t, err)
	require.Equal(t, entity.RunStatusRunning, gotLive.Status, "未超租约的运行不得被误杀")
}

// 租约必须按各行自己的超时判定：配了 30s 超时的运行若按默认 5min 算租约，会多挂几分钟才收敛
func TestSweepLeaseHonorsPerRunTimeout(t *testing.T) {
	svc := setupRunSvc(t, 1)

	req := baseReq()
	req.TimeoutSeconds = 30
	short := enqueueOK(t, svc, req)

	now := time.Now()
	claimed, err := svc.GetRepo().ClaimNextQueued(runTestCtx(), now, runInstanceId)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.Equal(t, short.Id, claimed.Id)

	// 240s 前开始：按自身租约 30+120=150s 已过期，但远小于默认基线 300+120=420s
	_, err = svc.GetRepo().UpdateColsByCond(runTestCtx(),
		base.NewPatch().Set("startedAt", now.Add(-4*time.Minute)),
		model.NewCond().EqStrict("id", short.Id))
	require.NoError(t, err)

	require.NoError(t, svc.SweepInterruptedRuns(runTestCtx()))

	got, err := svc.GetById(short.Id)
	require.NoError(t, err)
	require.Equal(t, entity.RunStatusFailed, got.Status, "租约必须用运行自身超时，而非服务默认值")
}

// 认领必须 FIFO（先入队先跑）：早期入队的运行不能被持续后来的任务挤后而饿死。
// 排序不显式写定时拿到的并不是最早一条，所以这里钉住顺序
func TestClaimNextQueuedIsFifo(t *testing.T) {
	svc := setupRunSvc(t, 1)

	first := enqueueOK(t, svc, baseReq())
	second := enqueueOK(t, svc, baseReq())
	third := enqueueOK(t, svc, baseReq())

	for _, want := range []*entity.Run{first, second, third} {
		claimed, err := svc.GetRepo().ClaimNextQueued(runTestCtx(), time.Now(), runInstanceId)
		require.NoError(t, err)
		require.NotNil(t, claimed)
		require.Equal(t, want.Id, claimed.Id, "认领顺序必须与入队顺序一致")
		require.Equal(t, entity.RunStatusRunning, claimed.Status)
	}

	none, err := svc.GetRepo().ClaimNextQueued(runTestCtx(), time.Now(), runInstanceId)
	require.NoError(t, err)
	require.Nil(t, none, "无待认领运行时返回 (nil, nil) 供 worker 结束本轮 drain")
}

// --- 有界工作池：并发上限 + 每个 Run 恰好执行一次（认领原子性） ---

func TestRunWorkerPoolBounded(t *testing.T) {
	const poolSize = 2
	const total = 6
	svc := setupRunSvc(t, poolSize)

	var (
		cur     int32
		maxSeen int32
		once    sync.Map // runNo -> 执行次数
	)
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		n := atomic.AddInt32(&cur, 1)
		for {
			old := atomic.LoadInt32(&maxSeen)
			if n <= old || atomic.CompareAndSwapInt32(&maxSeen, old, n) {
				break
			}
		}
		v, _ := once.LoadOrStore(req.Input, new(int32))
		atomic.AddInt32(v.(*int32), 1)

		time.Sleep(30 * time.Millisecond) // 制造重叠以观测并发上限
		atomic.AddInt32(&cur, -1)
		return &headless.RunResult{Status: headless.RunStatusSucceeded}, nil
	})

	svc.Start()
	ids := make([]uint64, 0, total)
	for i := 0; i < total; i++ {
		req := baseReq()
		req.Input = "task-input-" + strconv.Itoa(i) // 每 Run 唯一 input，用于校验各自恰好执行一次
		run := enqueueOK(t, svc, req)
		ids = append(ids, run.Id)
	}

	for _, id := range ids {
		waitStatus(t, svc, id, entity.RunStatusSucceeded)
	}
	require.LessOrEqual(t, atomic.LoadInt32(&maxSeen), int32(poolSize),
		"运行中并发不得超过工作池大小")

	// 认领原子性：并发 worker 不得重复执行同一 Run（每唯一 input 恰好一次）
	count := 0
	once.Range(func(k, v any) bool {
		count++
		require.Equal(t, int32(1), atomic.LoadInt32(v.(*int32)), "每个 Run 必须恰好执行一次")
		return true
	})
	require.Equal(t, total, count, "全部 Run 都应被执行")
}

// --- 未注册 headless 时不认领（启动竞态防护）+ 纯函数助手 ---

// 编排/评测父运行的租约基线必须按「整树预算」算：它们的 TimeoutSeconds 是单步语义，
// 按单步判定会在第 2 个阶段就被清扫器腰斩（CAS 抢输真实执行者的收尾，编排报告与
// token 汇总全部丢失）。旧测试抬不住是因为它不启动工作池，sweeper 从不参与那个用例。
func TestLeaseBaselineCoversMultiStageRuns(t *testing.T) {
	require.Equal(t, 300, runLeaseSeconds(&entity.Run{TimeoutSeconds: 300}), "普通运行按自身单步超时")

	orch := &entity.Run{TimeoutSeconds: 300, TriggerType: entity.TriggerTypeOrchestration}
	require.Equal(t, 300*(2+maxOrchestrationSteps), runLeaseSeconds(orch), "编排父运行需覆盖 planner+N步+reviewer")

	eval := &entity.Run{TimeoutSeconds: 300, TriggerRef: evalTriggerRefPrefix + "77"}
	require.Equal(t, 600, runLeaseSeconds(eval), "评测运行需覆盖重放+评审两段")

	// 未配超时时也必须乘阶段数，否则默认超时只保一段，长编排依旧被腰斩
	require.Equal(t, int(defaultRunTimeout/time.Second)*(2+maxOrchestrationSteps),
		runLeaseSeconds(&entity.Run{TriggerType: entity.TriggerTypeOrchestration}))

	require.Equal(t, 1, runStageCount(&entity.Run{TriggerRef: "webhook:7"}), "其他形态不得误放大")
}

func TestPureHelpers(t *testing.T) {
	require.Equal(t, headless.RunKindCron, triggerTypeToRunKind(entity.TriggerTypeCron))
	require.Equal(t, headless.RunKindEvent, triggerTypeToRunKind(entity.TriggerTypeEvent))
	require.Equal(t, headless.RunKindManual, triggerTypeToRunKind(entity.TriggerTypeManual))
	require.Equal(t, headless.RunKindManual, triggerTypeToRunKind(""))

	require.Equal(t, entity.RunStatusSucceeded, mapHeadlessStatus(headless.RunStatusSucceeded))
	require.Equal(t, entity.RunStatusCanceled, mapHeadlessStatus(headless.RunStatusCanceled))
	require.Equal(t, entity.RunStatusFailed, mapHeadlessStatus(headless.RunStatusFailed))
	require.Equal(t, entity.RunStatusFailed, mapHeadlessStatus(headless.RunStatusTimeout), "超时归入 failed")
	// 中断挂起必须落 failed：它不产生 error，若当成 succeeded 向下传播，
	// 提案受控执行会把「一次写操作都没发生」记成 executed（安全误导）
	require.Equal(t, entity.RunStatusFailed, mapHeadlessStatus(headless.RunStatusInterrupted), "无头运行里等待人工审批等于未执行，不得记成功")
	require.Equal(t, entity.RunStatusFailed, mapHeadlessStatus(""), "未知/空状态保守归 failed")

	require.True(t, entity.IsTerminal(entity.RunStatusSucceeded))
	require.True(t, entity.IsTerminal(entity.RunStatusFailed))
	require.True(t, entity.IsTerminal(entity.RunStatusCanceled))
	require.False(t, entity.IsTerminal(entity.RunStatusQueued))
	require.False(t, entity.IsTerminal(entity.RunStatusRunning))

	short := "boom"
	require.Equal(t, short, truncateRunErr(short))
	require.Equal(t, "", truncateRunErr(""))
	// 超长按 rune 截断到列宽
	require.Len(t, []rune(truncateRunErr(repeatRune('x', maxRunErrLen+50))), maxRunErrLen)
}

// --- GetPageList 过滤 ---

func TestGetPageList(t *testing.T) {
	svc := setupRunSvc(t, 1)
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusSucceeded}, nil
	})

	cron := baseReq()
	cron.TriggerType = entity.TriggerTypeCron
	cron.TriggerRef = "task-1"
	enqueueOK(t, svc, cron)
	manual := baseReq()
	manual.TriggerType = entity.TriggerTypeManual
	enqueueOK(t, svc, manual)

	page, err := svc.GetPageList(&entity.Run{TriggerType: entity.TriggerTypeCron}, model.PageParam{PageNum: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Equal(t, "task-1", page.List[0].TriggerRef)
}

// --- helpers ---

func tokUsage(p, c, tt int) *schema.TokenUsage {
	return &schema.TokenUsage{PromptTokens: p, CompletionTokens: c, TotalTokens: tt}
}

func repeatRune(r rune, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = r
	}
	return string(b)
}

// TestGetPageListOwnerFilterIsStrict 验证「按 owner 收窄」在真实仓储上真的生效。
//
// api 层测的是判定（谁需要收窄），这里测的是效果：收窄后的条件必须真的过滤掉别人的行，
// 而不是被基座的零值忽略逻辑吞掉（Eq 对零值不加条件，一旦 owner 传 0 就是全量返回）。
// 同时钉住「零值 owner = 不加过滤」这个基座语义，提醒 api 层不得拿零值当安全默认
func TestGetPageListOwnerFilterIsStrict(t *testing.T) {
	svc := setupRunSvc(t, 1)

	for _, ownerId := range []uint64{7, 7, 8} {
		req := baseReq()
		req.OwnerAccountId = ownerId
		req.OwnerUsername = fmt.Sprintf("u%d", ownerId)
		enqueueOK(t, svc, req)
	}

	own, err := svc.GetPageList(&entity.Run{OwnerAccountId: 8}, model.PageParam{PageNum: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), own.Total, "按 owner=8 收窄只能拿到自己的运行")
	require.Equal(t, uint64(8), own.List[0].OwnerAccountId)

	other, err := svc.GetPageList(&entity.Run{OwnerAccountId: 7}, model.PageParam{PageNum: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(2), other.Total, "owner=7 应有两条")
	for _, run := range other.List {
		require.Equal(t, uint64(7), run.OwnerAccountId)
	}

	// 零值 owner 不加过滤（基座语义）：这就是 api 层必须自己把无权限账号的 id 填进去的原因
	all, err := svc.GetPageList(&entity.Run{}, model.PageParam{PageNum: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(3), all.Total, "owner 为零值时条件被忽略、返回全量（所以收窄不能依赖零值）")
}

// TestGetPageListFiltersConversationAndParentRun 钉住「实体上声明了 form tag 的查询维度，仓储必须真的按它拼谓词」。
//
// 这两个维度此前都是死的：conversationId 缺 form tag（查询串绑不进查询实体），
// parentRunId 有 tag 但仓储没拼谓词（绑进来了也被忽略）。两种失败长得一模一样：
// 接口照返 200、列表返回全量，与「筛选生效但恰好没有匹配数据」在界面上不可区分。
// 前者由 entity 包的绑定用例守住，后者只能在这一层守——故两个维度都在此过一遍真实 SQL
func TestGetPageListFiltersConversationAndParentRun(t *testing.T) {
	svc := setupRunSvc(t, 1)
	fakeRunner(t, func(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		// 会话 id 从输入带出：入队时还没有会话，conversationId 由执行档返回后回写（与真实路径一致）
		convId, err := strconv.ParseUint(req.Input, 10, 64)
		require.NoError(t, err)
		return &headless.RunResult{Status: headless.RunStatusSucceeded, ConversationId: convId}, nil
	})

	svc.Start()
	// 编排 11 的两条子运行 + 一条无关运行；会话分别落在 9001/9002/9001
	for _, seed := range []struct {
		input  string
		parent uint64
	}{
		{"9001", 11},
		{"9002", 11},
		{"9001", 0},
	} {
		req := baseReq()
		req.Input = seed.input
		req.ParentRunId = seed.parent
		run := enqueueOK(t, svc, req)
		got := waitStatus(t, svc, run.Id, entity.RunStatusSucceeded)
		require.NotZero(t, got.ConversationId, "前置条件不成立：会话 id 未回写，后面的按会话筛会退化成零值不过滤")
	}

	byConv, err := svc.GetPageList(&entity.Run{ConversationId: 9001}, model.PageParam{PageNum: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(2), byConv.Total, "按会话筛运行应只拿到该会话产生的两条；拿到全量说明这个筛选维度是死的")
	for _, run := range byConv.List {
		require.Equal(t, uint64(9001), run.ConversationId, "结果里混进了别的会话的运行，等于没筛")
	}

	byParent, err := svc.GetPageList(&entity.Run{ParentRunId: 11}, model.PageParam{PageNum: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(2), byParent.Total, "按父运行筛编排子运行应只拿到这次编排的两条；拿到全量说明声明了却不生效")
	for _, run := range byParent.List {
		require.Equal(t, uint64(11), run.ParentRunId, "结果里混进了别的编排的子运行，等于没筛")
	}
}

// TestStopShutsDownPoolWithinTimeout 停机必须在超时内返回，且 Stop 幂等、可重启。
//
// 工作池的 worker 与租约 sweeper 共用同一个 stopCh/wg：任何一处没监听停止信号，
// Stop 就会永久阻塞在 wg.Wait()，表现为服务关不掉、滚动发布卡死；
// 而二次 Stop 若 close 已关闭的 channel 会 panic，panic 正好落在退出路径上。
// 这两个分支在接线停机钩子前从未被执行过（旧测试只测了未 Start 的实例）。
func TestStopShutsDownPoolWithinTimeout(t *testing.T) {
	svc := setupRunSvc(t, 2)

	stopWithin := func(label string) {
		t.Helper()
		done := make(chan struct{})
		go func() {
			defer close(done)
			svc.Stop()
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("%s：Stop 未在超时内返回，存在未监听停止信号的 goroutine", label)
		}
	}

	svc.Start()
	stopWithin("首次 Stop")
	stopWithin("重复 Stop（必须幂等而非 panic）")

	// 停机后可再次启动：若沿用已关闭的 stopCh，新 worker 一启动就收到停止信号退出
	svc.Start()
	require.True(t, svc.started, "Start 后应处于运行态")
	stopWithin("重启后 Stop")
	require.False(t, svc.started, "Stop 后必须置位，否则重复 Stop 会 panic")
}

// TestBothRunCreationPathsPersistRunRole 两条运行创建路径都必须把形态落进 Extra。
//
// 存在理由（真机抓过）：Enqueue（工作池）与 RunSync（审批回调后的受控执行/校验）各自拼一份
// entity.Run 字面量，只给 Enqueue 补 RunRole 时，提案的两条运行永远没有形态标签——不报错、
// 编译与单测全绿，只有界面上少一格标签。这类「同一入参两条落库路径」必须逐字段对齐并有用例钉住
func TestBothRunCreationPathsPersistRunRole(t *testing.T) {
	svc := setupRunSvc(t, 1)
	ctx := runTestCtx()
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: "ok"}, nil
	})

	queued := enqueueOK(t, svc, func() *EnqueueReq {
		r := baseReq()
		r.Title, r.RunRole = "池路径", entity.RunRoleOrchStep
		return r
	}())
	synced, err := svc.RunSync(ctx, func() *EnqueueReq {
		r := baseReq()
		r.Title, r.RunRole = "同步路径", entity.RunRoleExec
		return r
	}())
	require.NoError(t, err)
	require.NotNil(t, synced)

	// 读原始列而非实体字段：Extra 无反序列化实现，经实体读回随方言驱动而异（sqlite 读不回）
	rawExtra := func(id uint64) string {
		var got struct {
			Extra string `gorm:"column:extra"`
		}
		require.NoError(t, global.Db.Table("t_aiops_run").Select("extra").Where("id = ?", id).Take(&got).Error)
		return got.Extra
	}
	require.Contains(t, rawExtra(queued.Id), `"runRole":"orch_step"`, "Enqueue 路径丢了形态")
	require.Contains(t, rawExtra(synced.Id), `"runRole":"exec"`, "RunSync 路径丢了形态")
}

// TestFeedbackNoteReachesExperienceMemory 反馈备注必须出现在沉淀的记忆正文里。
//
// 这条链路此前**没有任何用例覆盖**，因此 `saveFeedbackMemory` 接了 note、还做了「空值→无备注」
// 归一化，却从没把它拼进 content ——Go 因为 note 在 if 里被引用过，不会报「声明未使用」，
// 编译/单测/界面全绿，只有运维填的理由被静默丢弃。而前端 prompt 对用户明说「将随经验沉淀进记忆系统」。
func TestFeedbackNoteReachesExperienceMemory(t *testing.T) {
	svc := setupRunSvc(t, 1)
	ctx := runTestCtx()

	store := &capturingMemoryStore{}
	prev := memory.DefaultStore
	memory.DefaultStore = store
	t.Cleanup(func() { memory.DefaultStore = prev })

	run := &entity.Run{
		RunNo: "rn-fb-note", Title: "慢查询巡检", Status: entity.RunStatusSucceeded,
		OwnerAccountId: 7, Input: "检查慢查询", Summary: "发现 3 条全表扫描",
		IdempotencyKey: "fb-note-1",
	}
	require.NoError(t, svc.GetRepo().Insert(ctx, run))

	const note = "结论方向对但漏了连接数上限这条风险"
	require.NoError(t, svc.Feedback(ctx, run.Id, entity.FeedbackUp, note))

	require.Len(t, store.saved, 1, "反馈应沉淀一条经验记忆")
	require.Contains(t, store.saved[0].Content, note, "备注必须进记忆正文，否则运维的人因信号丢失")
	require.Contains(t, store.saved[0].Tags, entity.FeedbackUp)

	// 无备注时走占位文案，不能留下空段落
	store2 := &capturingMemoryStore{}
	memory.DefaultStore = store2
	run2 := &entity.Run{RunNo: "rn-fb-nonote", Title: "t2", Status: entity.RunStatusFailed, OwnerAccountId: 7, IdempotencyKey: "fb-note-2"}
	require.NoError(t, svc.GetRepo().Insert(ctx, run2))
	require.NoError(t, svc.Feedback(ctx, run2.Id, entity.FeedbackDown, ""))
	require.Contains(t, store2.saved[0].Content, "无备注")
}

// capturingMemoryStore 记录被沉淀内容的假存储（只用于断言正文，不做检索）
type capturingMemoryStore struct{ saved []*memory.MemoryItem }

func (c *capturingMemoryStore) GetByUser(context.Context, string, []string) ([]*memory.MemoryItem, error) {
	return nil, nil
}
func (c *capturingMemoryStore) Save(_ context.Context, items []*memory.MemoryItem) error {
	c.saved = append(c.saved, items...)
	return nil
}
func (c *capturingMemoryStore) Delete(context.Context, string, []string) error { return nil }
func (c *capturingMemoryStore) Search(context.Context, string, string, int) ([]*memory.MemoryItem, error) {
	return nil, nil
}
