package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/infra/persistence"
	"mayfly-go/internal/ai/pkg/headless"
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

// setupOrchestrator 内存 sqlite + 真实仓储/RunService 装配编排器（不启动工作池：
// 编排的父/子运行都由 RunSync 在编排 goroutine 内同步执行，便于确定性断言）
func setupOrchestrator(t *testing.T) (*orchestratorAppImpl, *runServiceAppImpl) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		NamingStrategy:       gormx.MetaNamingStrategy(),
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(&entity.Run{}, &entity.Proposal{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	registerRepoOnce.Do(persistence.InitIoc)

	runSvc := &runServiceAppImpl{PoolSize: 1}
	runSvc.Repo = ioc.Get[repository.Run]()
	runSvc.signalCh = make(chan struct{}, 1)
	runSvc.stopCh = make(chan struct{})
	runSvc.inflight = make(map[uint64]context.CancelFunc)
	runSvc.cancelReq = make(map[uint64]struct{})
	t.Cleanup(runSvc.Stop)

	orch := &orchestratorAppImpl{runService: runSvc, proposalRepo: ioc.Get[repository.Proposal]()}
	return orch, runSvc
}

// 编排端到端：父运行与规划/步骤/审查子运行都必须是 orchestration 触发类型并带父归属，
// 且父运行汇总整棵树的 token（旧实现归为 manual、不汇总，实测父运行 Tokens=0 而子运行十万级，
// 执行历史既筛不出编排轨迹、成本也归属不清）
func TestOrchestrationTrailAndCostAggregation(t *testing.T) {
	orch, runSvc := setupOrchestrator(t)

	fakeRunner(t, func(_ context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		var out string
		switch req.ProfileId {
		case headless.ProfileExecutor:
			out = "unexpected executor run"
		case headless.ProfileReviewer:
			out = "目标已达成"
		default:
			// Planner 输出一个只读步骤；步骤运行输出结论（本用例只管轨迹与成本归属，
			// 提案谎行情形由 TestOrchestrationReportProposalsComeFromStore 专门覆盖）
			if strings.Contains(req.Input, "本步骤任务：") {
				out = "步骤结论：CPU 正常"
			} else {
				out = `[{"title":"查指标","action":"读监控指标","readonly":true}]`
			}
		}
		return &headless.RunResult{
			Status: headless.RunStatusSucceeded,
			Output: out,
			Usage:  &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120},
		}, nil
	})

	parent, err := orch.Orchestrate(context.Background(), &OrchestrateReq{
		Goal:           "排查负载",
		OwnerAccountId: 1,
		OwnerUsername:  "admin",
	})
	require.NoError(t, err)
	require.Equal(t, entity.TriggerTypeOrchestration, parent.TriggerType, "父运行触发类型必须是编排")

	require.Eventually(t, func() bool {
		got, gerr := runSvc.GetById(parent.Id)
		return gerr == nil && got != nil && entity.IsTerminal(got.Status)
	}, 10*time.Second, 50*time.Millisecond, "父运行未在预期时间内收尾")

	got, err := runSvc.GetById(parent.Id)
	require.NoError(t, err)
	require.Equal(t, entity.RunStatusSucceeded, got.Status)
	require.Contains(t, got.Summary, "最终结论", "父运行 summary 应承载编排报告")

	children, err := runSvc.ListByCond(model.NewCond().EqStrict("parentRunId", parent.Id))
	require.NoError(t, err)
	// planner + 1 步骤 + reviewer
	require.Len(t, children, 3, "编排轨迹必须按 parent_run_id 可整体筛出")
	var childTotal int64
	for _, c := range children {
		require.Equal(t, entity.TriggerTypeOrchestration, c.TriggerType, "子运行触发类型必须是编排")
		require.Equal(t, parent.Id, c.ParentRunId)
		childTotal += c.TotalTokens
	}
	require.Positive(t, childTotal)
	require.Equal(t, childTotal, got.TotalTokens, "父运行 token 必须等于子运行合计")
	require.Equal(t, childTotal, got.PromptTokens+got.CompletionTokens)
}

// TestOrchestrationReportProposalsComeFromStore 编排报告的提案清单必须由库决定，不能抬进步骤自述。
//
// 真机起因：某步骤在结论里写「并为该探测动作登记变更提案」，而那一轮根本没调 propose_remediation；
// 步骤自述被原样喂给 Reviewer 后，最终报告列了一条不存在的待审提案 —— 审批人按报告去列表找不到，
// 只会怀疑是列表漏了。本用例用同一条假 runner：步骤文本谎称已登记，而库里零提案
func TestOrchestrationReportProposalsComeFromStore(t *testing.T) {
	orch, runSvc := setupOrchestrator(t)

	fakeRunner(t, func(_ context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		var out string
		switch req.ProfileId {
		case headless.ProfileReviewer:
			// Reviewer 拿到的输入必须已含事实清单，否则它仍会只看到步骤自述
			require.Contains(t, req.Input, proposalsFactHeading, "Reviewer 输入必须带提案事实清单")
			require.Contains(t, req.Input, "0 条", "Reviewer 看到的清单必须说 0 条（库里确实没登记过）")
			out = "目标已达成"
		default:
			// 按「步骤运行特有的输入前缀」判分支，不比完整串：步骤 Input 里的任务文本取自计划的
			// action 字段，拿 title 去写死整串会永不命中，于是谎称已登记的那段输出根本没被产出，
			// 本判据就退化成空跑（上一版就踩了这个：两个 Index 都是 -1，Less 照样成立）
			if strings.Contains(req.Input, "本步骤任务：") {
				out = "步骤结论：CPU 正常，并已为该动作登记变更提案 #1，等待人工审批"
			} else {
				out = `[{"title":"查指标","action":"读监控指标","readonly":true}]`
			}
		}
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: out}, nil
	})

	parent, err := orch.Orchestrate(context.Background(), &OrchestrateReq{Goal: "排查负载", OwnerAccountId: 1, OwnerUsername: "admin"})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, gerr := runSvc.GetById(parent.Id)
		return gerr == nil && got != nil && entity.IsTerminal(got.Status)
	}, 10*time.Second, 50*time.Millisecond, "父运行未在预期时间内收尾")

	got, err := runSvc.GetById(parent.Id)
	require.NoError(t, err)
	report := got.Summary
	// 先确保谎言真的进了报告：没有它，下面的「当场证伪」断言会比较两个 -1 而恒成立
	lieAt := strings.Index(report, "并已为该动作登记变更提案")
	factAt := strings.Index(report, proposalsFactHeading)
	require.GreaterOrEqual(t, lieAt, 0, "步骤自述未按预期产出，本判据在空跑")
	require.GreaterOrEqual(t, factAt, 0, "父报告必须带提案事实清单小节")
	require.Contains(t, report, "0 条", "库里没登记过提案，报告必须说 0 条")
	// 事实清单必须落在步骤自述之后，阅读时才能当场证伪而不是先看到假数字
	require.Less(t, lieAt, factAt, "事实清单必须紧跟在步骤自述之后")
}

// TestProposalsFactText 清单生成的四种库态（无步骤 / 无提案 / 有提案 / 查不到）各自必须给出确定文案。
//
// 重点在最后一态：查询失败时不能静默省略那一节 —— 那等于让报告退回「只有步骤自述」的旧形态，
// 恰是本轮要修的洞；必须明确写出未经核实
func TestProposalsFactText(t *testing.T) {
	orch, _ := setupOrchestrator(t)
	ctx := context.Background()

	require.Contains(t, orch.proposalsFactText(ctx, nil), "未产出任何步骤运行", "没有步骤运行时就该确定地说没登记，不必进查询")
	require.Contains(t, orch.proposalsFactText(ctx, []uint64{77}), "0 条", "有步骤但库里无提案时报 0 条")

	require.NoError(t, orch.proposalRepo.Insert(ctx, &entity.Proposal{
		Title: "重启 nginx", Action: "MachineCommandExec", Risk: entity.ProposalRiskLow,
		Status: entity.ProposalStatusPending, OwnerAccountId: 1, RunId: 77,
	}))
	list := orch.proposalsFactText(ctx, []uint64{77})
	require.Contains(t, list, "重启 nginx", "已登记的提案必须逐条列出")
	require.Contains(t, list, entity.ProposalStatusPending)
	require.Contains(t, list, "来源运行 77", "必须带来源运行，审计能从报告回到那一步")
	require.NotContains(t, list, "0 条")

	// 不相关步骤 id 查不到：只能列出属于本编排的提案
	require.Contains(t, orch.proposalsFactText(ctx, []uint64{88}), "0 条", "不得把其它运行的提案计进本次编排")

	require.NoError(t, orch.proposalRepo.DeleteByCond(ctx, model.NewCond().EqStrict("runId", uint64(77))))
	require.Contains(t, orch.proposalsFactText(ctx, []uint64{77}), "0 条", "提案被清理后必须回到 0 条（不能缓存旧清单）")

	// 第四态：反查本身失败（这里直接抽掉表）。此时必须写明「未经核实」而不能静默省略那一节，
	// 否则报告退回「只有步骤自述」的旧形态——那正是本判据要堵的洞
	require.NoError(t, global.Db.Exec("DROP TABLE t_aiops_proposal").Error)
	failed := orch.proposalsFactText(ctx, []uint64{77})
	require.Contains(t, failed, "提案清单查询失败", "查不到必须明说，不能静默省略")
	require.Contains(t, failed, "未经核实", "要提醒读者此时步骤自述里的「已登记」不可信")
}
