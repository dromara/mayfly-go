package application

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/infra/persistence"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/ioc"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// govMemStore 内存急停开关存储（经 killSwitchStoreFn 接缝注入，隔离 sys 配置依赖；
// loadErr 可注入读失败场景验证 fail-open 语义）
type govMemStore struct {
	mu      sync.Mutex
	engaged bool
	loadErr error
}

func (m *govMemStore) Load() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.engaged, m.loadErr
}

func (m *govMemStore) Persist(_ context.Context, engaged bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engaged = engaged
	return nil
}

// govUseMemKillSwitch 覆写急停存储接缝为内存实现，用例结束自动复原原实现
func govUseMemKillSwitch(t *testing.T) *govMemStore {
	t.Helper()
	mem := &govMemStore{}
	prev := killSwitchStoreFn
	killSwitchStoreFn = func() killSwitchStore { return mem }
	t.Cleanup(func() { killSwitchStoreFn = prev })
	return mem
}

// govSetup 打开 sqlite 内存库、建治理相关四张表、装配绑定真实仓储的 governanceAppImpl。
// 与 RunService 单测同范式：真实仓储 + 内存库——Stats 的 SUM/AVG/CASE WHEN 聚合只有落到
// 真实 SQL 语义上才测得准；急停存储经接缝注入内存实现。
func govSetup(t *testing.T) *governanceAppImpl {
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
	require.NoError(t, db.AutoMigrate(&entity.Run{}, &entity.AgentTask{}, &entity.Trigger{}, &entity.Proposal{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	registerRepoOnce.Do(persistence.InitIoc)

	return &governanceAppImpl{
		runRepo:      ioc.Get[repository.Run](),
		taskRepo:     ioc.Get[repository.AgentTask](),
		triggerRepo:  ioc.Get[repository.Trigger](),
		proposalRepo: ioc.Get[repository.Proposal](),
	}
}

var govRunSeq int64

// govMkRun 构造一条合法 Run（RunNo/幂等键唯一，满足唯一索引约束；已带起止时间与耗时）
func govMkRun(status string, prompt, completion, total, costMs int64) *entity.Run {
	seq := atomic.AddInt64(&govRunSeq, 1)
	now := time.Now()
	return &entity.Run{
		RunNo:            fmt.Sprintf("rn-%d-%d", now.UnixNano(), seq),
		TriggerType:      entity.TriggerTypeManual,
		ProfileId:        headless.ProfileDiagnostician,
		Status:           status,
		OwnerAccountId:   1,
		OwnerUsername:    "tester",
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
		StartedAt:        &now,
		FinishedAt:       &now,
		CostMs:           costMs,
		IdempotencyKey:   fmt.Sprintf("auto:rn-%d-%d", now.UnixNano(), seq),
	}
}

func govInsertRun(t *testing.T, g *governanceAppImpl, run *entity.Run) {
	t.Helper()
	require.NoError(t, g.runRepo.Insert(context.Background(), run))
}

func TestGovernanceKillSwitch(t *testing.T) {
	g := govSetup(t)
	govUseMemKillSwitch(t)

	require.False(t, g.KillSwitch(), "默认未急停")

	require.NoError(t, g.SetKillSwitch(context.Background(), true))
	require.True(t, g.KillSwitch(), "拉起后应急停")

	require.NoError(t, g.SetKillSwitch(context.Background(), false))
	require.False(t, g.KillSwitch(), "解除后恢复自动化派发")
}

func TestGovernanceKillSwitchFailOpen(t *testing.T) {
	g := govSetup(t)
	mem := govUseMemKillSwitch(t)

	// 先拉起并让缓存生效
	require.NoError(t, g.SetKillSwitch(context.Background(), true))
	require.True(t, g.KillSwitch())

	// 存储读失败 + 缓存过期：fail-open 沿用上次缓存值，不误判、不 panic
	mem.mu.Lock()
	mem.loadErr = fmt.Errorf("store unavailable")
	mem.mu.Unlock()
	g.ksMu.Lock()
	g.ksLoaded = time.Time{}
	g.ksMu.Unlock()
	require.True(t, g.KillSwitch(), "读失败应沿用上次缓存（fail-open）")

	// 存储恢复后读到新值
	mem.mu.Lock()
	mem.loadErr = nil
	mem.engaged = false
	mem.mu.Unlock()
	g.ksMu.Lock()
	g.ksLoaded = time.Time{}
	g.ksMu.Unlock()
	require.False(t, g.KillSwitch(), "存储恢复后应读到新值")
}

func TestGovernanceStats(t *testing.T) {
	g := govSetup(t)
	govUseMemKillSwitch(t)
	ctx := context.Background()

	// 终态三条：succeeded(100/50/150, 1000ms) / failed(60/40/100, 500ms) / canceled(0/0/0, 0ms)
	govInsertRun(t, g, govMkRun(entity.RunStatusSucceeded, 100, 50, 150, 1000))
	govInsertRun(t, g, govMkRun(entity.RunStatusFailed, 60, 40, 100, 500))
	govInsertRun(t, g, govMkRun(entity.RunStatusCanceled, 0, 0, 0, 0))
	// 进行中两条：queued / running
	govInsertRun(t, g, govMkRun(entity.RunStatusQueued, 0, 0, 0, 0))
	govInsertRun(t, g, govMkRun(entity.RunStatusRunning, 0, 0, 0, 0))

	// 提案：2 待审批 + 1 已执行（不计入 pending）
	for i := 0; i < 2; i++ {
		require.NoError(t, g.proposalRepo.Insert(ctx, &entity.Proposal{
			Title:          fmt.Sprintf("p-%d", i),
			Action:         "ExecSql",
			Risk:           entity.ProposalRiskLow,
			Status:         entity.ProposalStatusPending,
			OwnerAccountId: 1,
		}))
	}
	require.NoError(t, g.proposalRepo.Insert(ctx, &entity.Proposal{
		Title:          "p-done",
		Action:         "ExecSql",
		Risk:           entity.ProposalRiskLow,
		Status:         entity.ProposalStatusExecuted,
		OwnerAccountId: 1,
	}))

	// 任务：1 启用 + 1 禁用
	require.NoError(t, g.taskRepo.Insert(ctx, &entity.AgentTask{
		Name: "task-on", Key: "task-key-on", Instruction: "do check",
		ProfileId: headless.ProfileDiagnostician, OwnerAccountId: 1, Status: entity.AgentTaskStatusEnable,
	}))
	require.NoError(t, g.taskRepo.Insert(ctx, &entity.AgentTask{
		Name: "task-off", Key: "task-key-off", Instruction: "do check",
		ProfileId: headless.ProfileDiagnostician, OwnerAccountId: 1, Status: entity.AgentTaskStatusDisable,
	}))

	// 触发器：1 启用 + 1 禁用
	require.NoError(t, g.triggerRepo.Insert(ctx, &entity.Trigger{
		Name: "trg-on", SourceType: "alert", Enabled: entity.TriggerEnabled,
	}))
	require.NoError(t, g.triggerRepo.Insert(ctx, &entity.Trigger{
		Name: "trg-off", SourceType: "alert", Enabled: entity.TriggerDisabled,
	}))

	stats, err := g.Stats(ctx, 30)
	require.NoError(t, err)

	require.Equal(t, int64(5), stats.RunTotal)
	require.Equal(t, int64(1), stats.RunByStatus[entity.RunStatusSucceeded])
	require.Equal(t, int64(1), stats.RunByStatus[entity.RunStatusFailed])
	require.Equal(t, int64(1), stats.RunByStatus[entity.RunStatusCanceled])
	require.Equal(t, int64(1), stats.RunByStatus[entity.RunStatusQueued])
	require.Equal(t, int64(1), stats.RunByStatus[entity.RunStatusRunning])
	require.Equal(t, int64(2), stats.ActiveRuns)
	// 成功率 = 1 / (1+1+1)
	require.InDelta(t, 1.0/3.0, stats.SuccessRate, 0.0001)
	// token 聚合只含终态三条（canceled 为 0）
	require.Equal(t, int64(160), stats.PromptTokens)
	require.Equal(t, int64(90), stats.CompletionTokens)
	require.Equal(t, int64(250), stats.TotalTokens)
	// 平均耗时只计「已终态且真跑过」的两条：canceled 这条 cost_ms=0（未起跑就被取消），
	// 纳入分母会把均值从 750 稀释到 500——急停清场几十条慢运行后看板会反而显示系统变快。
	// 旧断言写的正是被稀释后的值，等于把缺陷固化成回归基线，已按真实口径修正
	require.Equal(t, int64(750), stats.AvgCostMs)

	require.Equal(t, int64(2), stats.PendingProposals)
	require.Equal(t, int64(0), stats.StuckApproved, "待审批/已执行都不得泄进停滞计数")
	require.Equal(t, int(proposalStallAfter/time.Minute), stats.StallAfterMinutes, "静默期口径由服务端透出，界面不手抄常量")
	require.Equal(t, int64(1), stats.EnabledTasks)
	require.Equal(t, int64(1), stats.EnabledTriggers)
	require.False(t, stats.KillSwitch)
	require.Equal(t, 30, stats.WindowDays)
}

func TestGovernanceStatsEmpty(t *testing.T) {
	g := govSetup(t)
	govUseMemKillSwitch(t)

	stats, err := g.Stats(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(0), stats.RunTotal)
	require.Equal(t, int64(0), stats.ActiveRuns)
	require.Equal(t, int64(0), stats.PendingProposals)
	require.Equal(t, int64(0), stats.EnabledTasks)
	require.Equal(t, int64(0), stats.EnabledTriggers)
	require.Equal(t, 0.0, stats.SuccessRate, "无终态运行时成功率为 0")
	require.Equal(t, int64(0), stats.TotalTokens)
	require.Equal(t, 7, stats.WindowDays)
	// 状态分布五键齐全（进行中两类 + 终态三类），空库下也必须逐键落零：
	// 少一个键前端就多一个渲染不出标签的状态位
	require.Len(t, stats.RunByStatus, len(entity.ActiveRunStatuses)+len(entity.TerminalRunStatuses))
	require.Contains(t, stats.RunByStatus, entity.RunStatusQueued)
	require.Contains(t, stats.RunByStatus, entity.RunStatusSucceeded)
}

// TestGovernanceStuckApprovedMatchesStallPredicate 治理看板的停滞计数必须带静默期过滤，
// 且必须与停滞清扫的判据一致。
//
// 少了静默期就是纯假警报：exec_run_id 要等执行档返回终态才回写，所以一条**正常在途**的执行
// 同样是「approved 且 exec_run_id=0」，看板会把它计成停滞并打红——而假警报的代价是
// 真出问题（进程在执行前退出丢了单）时已经没人再看这个指标。
// 判据不一致则更隐蔽：看板报红的条数与清扫下一轮真正会去重投的条数不是同一批，
// 运维据看板做的判断从根上就不成立
func TestGovernanceStuckApprovedMatchesStallPredicate(t *testing.T) {
	g := govSetup(t)
	govUseMemKillSwitch(t)
	ctx := context.Background()
	staleAfter := proposalStallAfter + time.Hour

	seed := func(title, status string, execRunId uint64, backdate bool) uint64 {
		p := &entity.Proposal{
			Title: title, Action: "ExecSql", Risk: entity.ProposalRiskLow,
			Status: status, OwnerAccountId: 1, ExecRunId: execRunId,
		}
		require.NoError(t, g.proposalRepo.Insert(ctx, p))
		if backdate {
			backdateProposal(t, g.proposalRepo, p.Id, staleAfter)
		}
		return p.Id
	}

	// 唯一真停滞：已领取、无执行运行、且早已过静默期
	stalledId := seed("p-stalled", entity.ProposalStatusApproved, 0, true)
	// 正常在途：刚领取，执行档还在跑（exec_run_id 尚未回写）——绝不能计成停滞
	seed("p-inflight", entity.ProposalStatusApproved, 0, false)
	// 执行已落库：有关联就不属停滞
	seed("p-landed", entity.ProposalStatusApproved, 77, true)
	// 未审批：还没领取执行权
	seed("p-pending", entity.ProposalStatusPending, 0, true)
	// 终态：已执行完
	seed("p-executed", entity.ProposalStatusExecuted, 88, true)

	stats, err := g.Stats(ctx, 30)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats.StuckApproved,
		"只有「approved 且无执行运行且已过静默期」这一条算停滞；把在途的也算进去就是假警报")

	// 看板报的必须正是清扫会去动的那一批：同一判据、同一批 id
	swept, serr := g.proposalRepo.SelectByCond(stalledProposalCond(), "id")
	require.NoError(t, serr)
	require.Len(t, swept, 1, "清扫判据应恰好选中那一条真停滞的提案")
	require.Equal(t, stalledId, swept[0].Id, "看板计为停滞的与清扫判为停滞的必须是同一条，否则两个权威互相矛盾")

	require.Positive(t, stats.StallAfterMinutes, "静默期为 0 会把所有在途执行都判成停滞")
}
