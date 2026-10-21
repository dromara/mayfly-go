package application

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/infra/persistence"
	"mayfly-go/internal/ai/pkg/headless"
	aitools "mayfly-go/internal/ai/tools"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupProposalGateway 打开 sqlite 内存库（Run + Proposal 两表），装配一个绑定真实仓储、
// 已注入 RunService 并已启动受控执行异步体的 ProposalGateway。RunService 工作池「不启动」：
// RunSync 在异步体的 goroutine 内同步执行（不经工作池），headless.Runner 由用例注入 fake 隔离真实 LLM。
// flow 应用（procdef/procinst）经 stubFlow 覆写间接层注入最小 fake，无需装配整个 flow 模块。
func setupProposalGateway(t *testing.T) (*proposalGatewayImpl, *runServiceAppImpl) {
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

	gw := &proposalGatewayImpl{}
	gw.Repo = ioc.Get[repository.Proposal]()
	gw.runService = runSvc
	// 启动受控执行异步体（与生产 Init 一致），否则投递会被拒
	gw.StartExecutionWorker()
	t.Cleanup(gw.StopExecutionWorker)
	return gw, runSvc
}

// waitProposalStatus 等待提案进入期望状态（受控执行已改为异步，断言需轮询而非假定已同步跑完）
func waitProposalStatus(t *testing.T, gw *proposalGatewayImpl, id uint64, want string) *entity.Proposal {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var got *entity.Proposal
	for time.Now().Before(deadline) {
		p, err := gw.GetById(id)
		if err == nil && p != nil && p.Status == want {
			return p
		}
		got = p
		time.Sleep(10 * time.Millisecond)
	}
	status := "<nil>"
	if got != nil {
		status = got.Status
	}
	t.Fatalf("proposal %d did not reach status %q in time (last=%q)", id, want, status)
	return nil
}

// stubFlow 覆写 flow 应用解析间接层为返回给定 fake 的函数（用例结束自动复原）。
// 传 nil 表示「flow 应用未就绪」，用于覆盖 fail-soft / no-op 分支。
func stubFlow(t *testing.T, procdef procdefFinder, procinst procinstRunner) {
	t.Helper()
	prevDef, prevInst := flowProcdefFn, flowProcinstFn
	flowProcdefFn = func() procdefFinder { return procdef }
	flowProcinstFn = func() procinstRunner { return procinst }
	t.Cleanup(func() { flowProcdefFn, flowProcinstFn = prevDef, prevInst })
}

// fakeProcdef 最小 procdefFinder：GetByCond 命中时把预置流程定义回填到入参指针
type fakeProcdef struct {
	def *flowentity.Procdef // nil → 返回 not found（模拟未配置审批流）
}

func (f *fakeProcdef) GetByCond(cond any) error {
	if f.def == nil {
		return errors.New("procdef not found")
	}
	if target, ok := cond.(*flowentity.Procdef); ok {
		*target = *f.def
	}
	return nil
}

// fakeProcinst 最小 procinstRunner：StartProc 返回预置 id 的工单（或错误），GetById 从 byId 查，
// RecordBizHandleResult 捕获终态回写供断言
type fakeProcinst struct {
	startId  uint64
	startErr error
	byId     map[uint64]*flowentity.Procinst
	started  []*flowdto.StarProc // 捕获发起的工单参数供断言
	bizRecs  []fakeBizRecord     // 捕获终态回写（工单 id/状态/结果 JSON）
}

// fakeBizRecord 一次 RecordBizHandleResult 调用的入参快照
type fakeBizRecord struct {
	ProcInstId uint64
	BizStatus  flowentity.ProcinstBizStatus
	HandleRes  string
}

func (f *fakeProcinst) RecordBizHandleResult(_ context.Context, procinstId uint64, bizStatus flowentity.ProcinstBizStatus, handleRes string) error {
	f.bizRecs = append(f.bizRecs, fakeBizRecord{procinstId, bizStatus, handleRes})
	return nil
}

func (f *fakeProcinst) StartProc(_ context.Context, _ uint64, reqParam *flowdto.StarProc) (*flowentity.Procinst, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}
	f.started = append(f.started, reqParam)
	pi := &flowentity.Procinst{Status: flowentity.ProcinstStatusActive}
	pi.Id = f.startId
	return pi, nil
}

func (f *fakeProcinst) GetById(id uint64, _ ...string) (*flowentity.Procinst, error) {
	if pi, ok := f.byId[id]; ok {
		return pi, nil
	}
	return nil, errors.New("procinst not found")
}

// procinstWithStatus 构造一个指定 id/状态的流程实例值（BizHandleParam.Procinst 为值类型）
func procinstWithStatus(id uint64, status flowentity.ProcinstStatus) flowentity.Procinst {
	pi := flowentity.Procinst{Status: status}
	pi.Id = id
	return pi
}

func procinstPtr(id uint64, status flowentity.ProcinstStatus) *flowentity.Procinst {
	pi := &flowentity.Procinst{Status: status}
	pi.Id = id
	return pi
}

// enabledProcdef 一个启用状态、指定 id 的审批流定义
func enabledProcdef(id uint64) *flowentity.Procdef {
	def := &flowentity.Procdef{Status: flowentity.ProcdefStatusEnable, DefKey: defaultRemediationProcDefKey}
	def.Id = id
	return def
}

// --- Record：登记校验 + fail-soft ---

func TestProposalRecordFailClosed(t *testing.T) {
	gw, _ := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	ctx := context.Background()

	_, err := gw.Record(ctx, nil)
	require.Error(t, err, "nil 草稿须拒绝")

	_, err = gw.Record(ctx, &aitools.ProposalDraft{OwnerAccountId: 1, Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Reason: "r"})
	require.Error(t, err, "缺 action 须拒绝")

	_, err = gw.Record(ctx, &aitools.ProposalDraft{Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Reason: "r"})
	require.Error(t, err, "无归属身份须 fail-closed 拒绝")
}

// TestProposalRecordRejectsMalformedParams 登记时就必须校验参数形状，不能只放在下发受控执行许可时。
//
// 端到端实测到模型提交的 params 少了必填 remark（形如 {"authCertName":"…","command":"true"}）
// 却登记成功并进审批流：审批人看不到「执行目的」这个关键判断依据，变更还要等执行那步
// 才被拒，白走一轮审批。
// 必须断言失败原因：只断 Error 会被无关校验顺带满足（同类变异存活过一次）
func TestProposalRecordRejectsMalformedParams(t *testing.T) {
	gw, _ := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	ctx := context.Background()

	// 真实形态：键名都对但缺必填 remark
	_, err := gw.Record(ctx, &aitools.ProposalDraft{
		Title: "在 aliyun 执行 true", Action: "MachineCommandExec",
		Params: `{"authCertName":"c1","command":"true"}`, Risk: "low", Reason: "r",
		OwnerAccountId: 7, OwnerUsername: "ops",
	})
	require.Error(t, err, "缺必填项的提案不得入库")
	require.Contains(t, err.Error(), "miss parameter")
	require.Contains(t, err.Error(), "remark")

	// 历史坏提案的真实形态：自造键名 machineId/cmd（当年它一路入库并被批准，执行时才发现跑不了）
	_, err = gw.Record(ctx, &aitools.ProposalDraft{
		Title: "坏提案", Action: "MachineCommandExec",
		Params: `{"machineId":1,"cmd":"true"}`, Risk: "low", Reason: "r",
		OwnerAccountId: 7, OwnerUsername: "ops",
	})
	require.Error(t, err, "未知参数键的提案不得入库")
	require.Contains(t, err.Error(), "unknown parameter")

	// 形状齐备时正常登记：上面两条必须是因为参数形状被拒，而不是测试构造本身有问题
	rec, err := gw.Record(ctx, &aitools.ProposalDraft{
		Title: "好提案", Action: "MachineCommandExec",
		Params: `{"authCertName":"c1","command":"true","remark":"探活"}`, Risk: "low", Reason: "r",
		OwnerAccountId: 7, OwnerUsername: "ops",
	})
	require.NoError(t, err)
	require.NotZero(t, rec.ProposalId)
}

// TestProposalRecordSkipsUndeclaredParamShapes 未声明入参形状的工具仍可登记，不因新增校验误伤外部工具。
//
// 旧用例拿 DbQueryData 作样本，但它是能力表里的只读工具——M3 之后被「不能作 action」拦住，
// 与「跳过形状校验」无关。改用显式声明为 Write 的探针工具：只测「形状未声明 → 跳过」这一条。
//
// 探针残留在包级能力表里是无害的：全仓只有 DangerousToolNames() 遍历该表且按 Dangerous 过滤，
// 本探针是 Write，不会进任何「高危工具必须声明形状」类守卫的集合。
func TestProposalRecordSkipsUndeclaredParamShapes(t *testing.T) {
	const probe = "__test_proposal_probe_tool__"
	// 声明为可变工具：不声明则 CapabilityOf 缺省也返 Write，但「缺省」与「已声明」在 IsDeclaredTool
	// 上不同（未声明不是合法工具名，会被直接拒），所以这一步是必需的
	aitools.DeclareCapability(probe, aitools.CapabilityWrite)

	gw, _ := setupProposalGateway(t)
	stubFlow(t, nil, nil)

	rec, err := gw.Record(context.Background(), &aitools.ProposalDraft{
		Title: "外部写工具", Action: probe, Params: `{"anything":1}`, Risk: "low", Reason: "r",
		OwnerAccountId: 7, OwnerUsername: "ops",
	})
	require.NoError(t, err, "未声明形状的可变工具应跳过参数校验")
	require.NotZero(t, rec.ProposalId)
}

func TestProposalRecordNoProcdef(t *testing.T) {
	gw, _ := setupProposalGateway(t)
	stubFlow(t, nil, nil) // 未绑定审批流

	rec, err := gw.Record(context.Background(), &aitools.ProposalDraft{
		Title: "加索引", Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"create index","remark":"消除慢查询"}`, Risk: "high", Reason: "慢查询",
		OwnerAccountId: 7, OwnerUsername: "ops",
	})
	require.NoError(t, err, "未绑定审批流不阻断登记（fail-soft）")
	require.NotNil(t, rec)
	require.NotZero(t, rec.ProposalId)
	require.Equal(t, entity.ProposalStatusPending, rec.Status)
	require.Zero(t, rec.ProcInstId, "未发起工单")
	require.NotEmpty(t, rec.Message)

	saved, gerr := gw.GetById(rec.ProposalId)
	require.NoError(t, gerr)
	require.Equal(t, entity.ProposalStatusPending, saved.Status)
	require.Equal(t, "ExecSql", saved.Action)
	require.Equal(t, uint64(7), saved.OwnerAccountId)
}

func TestProposalRecordStartsApproval(t *testing.T) {
	gw, _ := setupProposalGateway(t)
	inst := &fakeProcinst{startId: 9001}
	stubFlow(t, &fakeProcdef{def: enabledProcdef(555)}, inst)

	rec, err := gw.Record(context.Background(), &aitools.ProposalDraft{
		Title: "重启服务", Action: "MachineCommandExec", Params: `{"authCertName":"c1","command":"systemctl restart x","remark":"重启服务"}`,
		Risk: "critical", Reason: "进程假死", OwnerAccountId: 3, OwnerUsername: "ops3",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(9001), rec.ProcInstId)
	require.Equal(t, entity.ProposalStatusPending, rec.Status)

	require.Len(t, inst.started, 1, "应发起一次审批工单")
	sp := inst.started[0]
	require.Equal(t, AgentRemediationFlowBizType, sp.BizType)
	require.Equal(t, bizKeyPrefix+strconv.FormatUint(rec.ProposalId, 10), sp.BizKey)
	require.NotEmpty(t, sp.BizForm, "工单须携带提案详情表单")

	saved, gerr := gw.GetById(rec.ProposalId)
	require.NoError(t, gerr)
	require.Equal(t, uint64(9001), saved.ProcInstId, "回写工单实例 id")
	require.Equal(t, uint64(555), saved.ProcDefId, "回写工单定义 id")
}

func TestProposalRecordStartProcFailSoft(t *testing.T) {
	gw, _ := setupProposalGateway(t)
	inst := &fakeProcinst{startErr: errors.New("flow down")}
	stubFlow(t, &fakeProcdef{def: enabledProcdef(555)}, inst)

	rec, err := gw.Record(context.Background(), &aitools.ProposalDraft{
		Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Reason: "r", OwnerAccountId: 1,
	})
	require.NoError(t, err, "发起工单失败不回滚提案（fail-soft）")
	require.NotNil(t, rec)
	require.NotZero(t, rec.ProposalId)
	require.Zero(t, rec.ProcInstId)
	require.NotEmpty(t, rec.Message)

	saved, gerr := gw.GetById(rec.ProposalId)
	require.NoError(t, gerr)
	require.Equal(t, entity.ProposalStatusPending, saved.Status, "提案仍在库、待审批")
}

// procinstWithBizKey 携带业务键的已完工单（用于 proc_inst_id 未绑定时按键反查的兜底路径）
func procinstWithBizKey(id uint64, bizKey string) flowentity.Procinst {
	pi := flowentity.Procinst{Status: flowentity.ProcinstStatusCompleted, BizKey: bizKey}
	pi.Id = id
	return pi
}

// --- FlowBizHandle：审批回调领取后投递异步执行，执行体完成受控执行 + 校验闭环 ---

func TestProposalFlowBizHandleExecutes(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	// FlowBizHandle 经 param 拿 procinst；fake 工单应用只服务于终态结果回写断言
	inst := &fakeProcinst{}
	stubFlow(t, nil, inst)
	var execGrant []string
	fakeRunner(t, func(_ context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		out := "executed: index created"
		switch req.ProfileId {
		case headless.ProfileReviewer:
			out = "verified: slow query resolved"
		case headless.ProfileExecutor:
			execGrant = req.AllowedToolNames
			// 执行档必须收到「只允许已批准那一个工具」的一次性许可（机制级约束，不只是提示词）
			require.Equal(t, []string{"ExecSql"}, req.AllowedToolNames, "执行档必须收到提案 action 的一次性许可")
		}
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: out}, nil
	})

	p := &entity.Proposal{
		Title: "加索引", Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"create index","remark":"消除慢查询"}`, Risk: "high", Reason: "慢查询",
		Status: entity.ProposalStatusPending, OwnerAccountId: 7, OwnerUsername: "ops", ProcInstId: 9001,
	}
	require.NoError(t, gw.Save(context.Background(), p))

	res, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithStatus(9001, flowentity.ProcinstStatusCompleted),
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	// 审批回调不再同步跑 LLM：领取即投递，工单当次请求内拿到「已受理」
	require.Equal(t, "queued", res.(map[string]any)["execution"], "回调立即返回，执行在异步体里跑")
	require.Equal(t, entity.ProposalStatusApproved, res.(map[string]any)["status"])

	saved := waitProposalStatus(t, gw, p.Id, entity.ProposalStatusExecuted)
	require.NotZero(t, saved.ExecRunId)
	require.NotZero(t, saved.VerifyRunId)
	require.Contains(t, saved.ExecResult, "executed")
	require.Contains(t, saved.VerifyConclusion, "verified")
	// 真实执行结果必须回写审批工单：只存「已受理」回执时，审批人回到工单看不出变更跑没跑成
	require.Len(t, inst.bizRecs, 1, "executed 终态应恰好回写一次工单")
	rec := inst.bizRecs[0]
	require.Equal(t, uint64(9001), rec.ProcInstId)
	require.Equal(t, flowentity.ProcinstBizStatusSuccess, rec.BizStatus)
	require.Contains(t, rec.HandleRes, "executed")
	require.Contains(t, rec.HandleRes, "verified: slow query resolved", "工单里的处理结果必须是真实执行/校验输出而非入队回执")
	require.NotContains(t, rec.HandleRes, "queued", "终态回写不得仍停在受理回执")
	require.Equal(t, []string{"ExecSql"}, execGrant, "许可必须从 Run 行读回并一路传到 headless")
	// 负责人账号名已收敛进 extra：纯列更新（状态流转）不得连带抹掉它，读回仍须经 AfterFind 还原
	require.Equal(t, "ops", saved.OwnerUsername, "负责人账号名在多次状态流转后仍须读回")

	// 闭环产出 exec + verify 两条 Run，档位正确
	cnt, cerr := runSvc.CountByCond(&entity.Run{})
	require.NoError(t, cerr)
	require.Equal(t, int64(2), cnt)

	execRun, _ := runSvc.GetById(saved.ExecRunId)
	require.Equal(t, headless.ProfileExecutor, execRun.ProfileId)
	require.Equal(t, entity.RunStatusSucceeded, execRun.Status)
	// 许可随运行落库（运行自持）：同步/异步两条路径同源
	require.Equal(t, []string{"ExecSql"}, execRun.AllowedToolNames())
	verifyRun, _ := runSvc.GetById(saved.VerifyRunId)
	require.Equal(t, headless.ProfileReviewer, verifyRun.ProfileId)
	require.Equal(t, entity.RunStatusSucceeded, verifyRun.Status)
}

// 审批回写 procInstId 失败（关联未落库）时，回调必须能按工单业务键反查到提案并自愈。
// 否则「工单批了、提案永远 pending、三处都说没问题」是本项目最难发现的一类失效
func TestProposalFlowBizHandleLocatesByBizKey(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: "done"}, nil
	})

	// ProcInstId=0：模拟 Record 里那次回写失败
	p := &entity.Proposal{Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Risk: "high", Reason: "r",
		Status: entity.ProposalStatusPending, OwnerAccountId: 1}
	require.NoError(t, gw.Save(context.Background(), p))

	_, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithBizKey(9100, bizKeyPrefix+strconv.FormatUint(p.Id, 10)),
	})
	require.NoError(t, err)

	saved := waitProposalStatus(t, gw, p.Id, entity.ProposalStatusExecuted)
	require.Equal(t, uint64(9100), saved.ProcInstId, "按业务键找到后必须补回关联")
	cnt, _ := runSvc.CountByCond(&entity.Run{})
	require.Equal(t, int64(2), cnt, "兜底路径同样走完执行+校验")
}

// action 不是可调用工具名时必须拒绝启动受控执行（而不是放开全量工具）
func TestProposalNonToolActionRefusesExecution(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	runs := 0
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		runs++
		return &headless.RunResult{Status: headless.RunStatusSucceeded}, nil
	})

	p := &entity.Proposal{Action: "restart-service", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Risk: "high", Reason: "r",
		Status: entity.ProposalStatusPending, OwnerAccountId: 1, ProcInstId: 9200}
	require.NoError(t, gw.Save(context.Background(), p))

	_, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithStatus(9200, flowentity.ProcinstStatusCompleted),
	})
	require.NoError(t, err)

	saved := waitProposalStatus(t, gw, p.Id, entity.ProposalStatusFailed)
	require.Contains(t, saved.Err, "restart-service", "失败原因要能直接指出是哪个动作解析不出工具")
	require.Zero(t, saved.ExecRunId)
	require.Equal(t, 0, runs, "解析不出许可就不得启动任何执行运行")
	cnt, _ := runSvc.CountByCond(&entity.Run{})
	require.Equal(t, int64(0), cnt)
}

func TestProposalFlowBizHandleNonCompleted(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusSucceeded}, nil
	})

	p := &entity.Proposal{Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Status: entity.ProposalStatusPending, OwnerAccountId: 1, ProcInstId: 9002}
	require.NoError(t, gw.Save(context.Background(), p))

	res, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithStatus(9002, flowentity.ProcinstStatusActive),
	})
	require.NoError(t, err)
	require.Nil(t, res, "非完成状态不处理")

	saved, _ := gw.GetById(p.Id)
	require.Equal(t, entity.ProposalStatusPending, saved.Status, "未流转")
	cnt, _ := runSvc.CountByCond(&entity.Run{})
	require.Equal(t, int64(0), cnt, "未产运行")
}

func TestProposalFlowBizHandleIdempotent(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusSucceeded}, nil
	})

	p := &entity.Proposal{Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Status: entity.ProposalStatusExecuted, OwnerAccountId: 1, ProcInstId: 9003}
	require.NoError(t, gw.Save(context.Background(), p))

	res, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithStatus(9003, flowentity.ProcinstStatusCompleted),
	})
	require.NoError(t, err)
	require.Equal(t, AckMessageAlreadyProcessed, res.(map[string]any)[AckKeyMessage])

	cnt, _ := runSvc.CountByCond(&entity.Run{})
	require.Equal(t, int64(0), cnt, "终态短路，不重复产运行")
}

// CAS 落败的两条收敛路径：上一次领取后崩在执行前（approved 且无执行运行）要重投自愈；
// 已在执行/已终态则不得重复执行（幂等键与状态机共同保证）
func TestProposalFlowBizHandleCasLose(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: "ok"}, nil
	})

	// 情形 A：approved 但 exec_run_id=0（领取后崩在执行之前）→ 重新投递，不永远悬挂
	orphan := &entity.Proposal{Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Risk: "high", Reason: "r",
		Status: entity.ProposalStatusApproved, OwnerAccountId: 1, ProcInstId: 9004}
	require.NoError(t, gw.Save(context.Background(), orphan))

	res, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithStatus(9004, flowentity.ProcinstStatusCompleted),
	})
	require.NoError(t, err)
	require.Equal(t, "requeued", res.(map[string]any)["execution"], "悬挂的 approved 必须重投")
	waitProposalStatus(t, gw, orphan.Id, entity.ProposalStatusExecuted)

	// 情形 B：已领取且已开始执行（exec_run_id 已写）→ 幂等返回，不重复执行
	claimed := &entity.Proposal{Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Risk: "high", Reason: "r",
		Status: entity.ProposalStatusApproved, OwnerAccountId: 1, ProcInstId: 9006, ExecRunId: 777}
	require.NoError(t, gw.Save(context.Background(), claimed))

	res2, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithStatus(9006, flowentity.ProcinstStatusCompleted),
	})
	require.NoError(t, err)
	require.Equal(t, AckMessageAlreadyClaimed, res2.(map[string]any)[AckKeyMessage])

	// 情形 A 产出 exec+verify 两条；情形 B 一条也不新增
	cnt, _ := runSvc.CountByCond(&entity.Run{})
	require.Equal(t, int64(2), cnt, "CAS 落败但已在执行的提案不得重复产运行")
}

func TestProposalExecFailureMarksFailed(t *testing.T) {
	gw, _ := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		return &headless.RunResult{Status: headless.RunStatusFailed}, errors.New("exec boom")
	})

	p := &entity.Proposal{
		Action: "ExecSql", Params: `{"dbId":5,"dbName":"","sql":"select 1","remark":"e2e"}`, Reason: "r", Risk: "high",
		Status: entity.ProposalStatusPending, OwnerAccountId: 1, ProcInstId: 9005,
	}
	require.NoError(t, gw.Save(context.Background(), p))

	res, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithStatus(9005, flowentity.ProcinstStatusCompleted),
	})
	require.NoError(t, err, "执行失败不在审批回调里回传（已异步），靠提案终态与运行记录暴露")
	require.Equal(t, "queued", res.(map[string]any)["execution"])

	saved := waitProposalStatus(t, gw, p.Id, entity.ProposalStatusFailed)
	require.NotEmpty(t, saved.Err)
	require.NotZero(t, saved.ExecRunId, "关联失败的执行运行以便排查")
	require.Zero(t, saved.VerifyRunId, "执行失败不进入校验")
}

// 受控执行里工具报错（run 本身成功）必须把提案判成 failed:「变更没发生」不能被记成「已执行」。
// 同时工具错误要随运行落库（Extra）并写进提案 err，否则只能从模型文本里找线索
func TestProposalExecToolFailureMarksFailed(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	inst := &fakeProcinst{}
	stubFlow(t, nil, inst)
	fakeRunner(t, func(_ context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		if req.ProfileId == headless.ProfileExecutor {
			return &headless.RunResult{
				Status:     headless.RunStatusSucceeded,
				Output:     "执行状态：失败，工具返回 Access denied",
				ToolErrors: []string{"MachineCommandExec: [tool error] Access denied for user"},
			}, nil
		}
		t.Error("工具失败后不应再发起校验运行")
		return &headless.RunResult{Status: headless.RunStatusSucceeded}, nil
	})

	// action 必须是可变工具（只读工具已被能力闸门拦住）：本用例测的是「工具报错→提案 failed」，
	// 不是测 action 准入，因此用真实可变工具 + 合形参数
	p := &entity.Proposal{Action: "MachineCommandExec", Params: `{"authCertName":"c1","command":"true","remark":"e2e"}`, Risk: "low", Reason: "r",
		Status: entity.ProposalStatusPending, OwnerAccountId: 1, ProcInstId: 9300}
	require.NoError(t, gw.Save(context.Background(), p))

	_, err := gw.FlowBizHandle(context.Background(), &flowapp.BizHandleParam{
		Procinst: procinstWithStatus(9300, flowentity.ProcinstStatusCompleted),
	})
	require.NoError(t, err)

	saved := waitProposalStatus(t, gw, p.Id, entity.ProposalStatusFailed)
	require.Contains(t, saved.Err, "Access denied", "提案错误原因必须是工具级失败原文")
	require.NotZero(t, saved.ExecRunId)
	require.Zero(t, saved.VerifyRunId, "执行未成功不进入校验环节")
	// 失败终态同样回写工单，且把 biz_status 从回调时的「处理成功」改判为「处理失败」：
	// 否则工单列表说批完一切正常而变更根本没落地
	require.Len(t, inst.bizRecs, 1)
	require.Equal(t, flowentity.ProcinstBizStatusFail, inst.bizRecs[0].BizStatus)
	require.Contains(t, inst.bizRecs[0].HandleRes, "Access denied")

	execRun, gerr := runSvc.GetById(saved.ExecRunId)
	require.NoError(t, gerr)
	require.Equal(t, entity.RunStatusSucceeded, execRun.Status, "run 自身仍是成功（它确实跑完了）")
	require.Len(t, execRun.ToolErrors(), 1, "工具级失败必须随运行自持")
	require.Contains(t, execRun.ToolErrors()[0], "MachineCommandExec", "工具级失败原文必须随运行落库（Extra）")
}

// --- SyncProcStatus：按工单终态惰性对账 ---

func TestProposalSyncProcStatus(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	inst := &fakeProcinst{byId: map[uint64]*flowentity.Procinst{
		8001: procinstPtr(8001, flowentity.ProcinstStatusCancelled),
		8002: procinstPtr(8002, flowentity.ProcinstStatusActive),
		8003: procinstPtr(8003, flowentity.ProcinstStatusBack),
		8004: procinstPtr(8004, flowentity.ProcinstStatusTerminated),
		8005: procinstPtr(8005, flowentity.ProcinstStatusCompleted),
		8006: procinstPtr(8006, flowentity.ProcinstStatusSuspended),
	}}
	stubFlow(t, nil, inst)

	mk := func(procInstId uint64) *entity.Proposal {
		return &entity.Proposal{Action: "a", Status: entity.ProposalStatusPending, OwnerAccountId: 1, ProcInstId: procInstId}
	}
	pCancelled, pActive, pBack := mk(8001), mk(8002), mk(8003)
	pTerminated, pCompleted, pSuspended := mk(8004), mk(8005), mk(8006)
	pNoProc := &entity.Proposal{Action: "a", Status: entity.ProposalStatusPending, OwnerAccountId: 1}
	all := []*entity.Proposal{pCancelled, pActive, pBack, pTerminated, pCompleted, pSuspended, pNoProc}
	for _, p := range all {
		require.NoError(t, gw.Save(context.Background(), p))
	}

	gw.SyncProcStatus(context.Background(), all)

	require.Equal(t, entity.ProposalStatusRejected, pCancelled.Status, "工单取消 → rejected（入参就地更新）")
	require.Equal(t, entity.ProposalStatusRejected, pTerminated.Status, "工单终止（审批页的「拒绝」就是它）→ rejected")
	require.Equal(t, entity.ProposalStatusPending, pActive.Status, "进行中 → 保持 pending")
	require.Equal(t, entity.ProposalStatusPending, pBack.Status, "退回可重提 → 保持 pending")
	require.Equal(t, entity.ProposalStatusPending, pSuspended.Status, "挂起可恢复 → 保持 pending")
	require.Equal(t, entity.ProposalStatusPending, pNoProc.Status, "未绑工单 → 跳过")

	// 工单已完成而提案仍在 pending：审批已成事实，不对账就会留下「工单说已批准、提案说待审批」
	// 的永久矛盾（实测到一条提案就这么停了五天，而它的工单回执里写着 execution=queued）
	require.Equal(t, entity.ProposalStatusApproved, pCompleted.Status, "工单已完成 → approved，交给停滞清扫接手")

	saved, _ := gw.GetById(pCancelled.Id)
	require.Equal(t, entity.ProposalStatusRejected, saved.Status, "对账结果落库")
	savedCompleted, _ := gw.GetById(pCompleted.Id)
	require.Equal(t, entity.ProposalStatusApproved, savedCompleted.Status, "完成对账同样要落库（否则下一次读取又重来一遍）")
	require.Zero(t, savedCompleted.ExecRunId, "对账不得凭空造出执行运行关联")

	// 对账跑在读路径（提案列表）上，绝不得产生运行：那等于一次 GET 请求就对目标机器下发命令
	cnt, cerr := runSvc.CountByCond(&entity.Run{})
	require.NoError(t, cerr)
	require.Equal(t, int64(0), cnt, "读路径对账只改状态，不得产出任何运行")
}

// allProcinstStatuses 从枚举反查全部工单状态值（int8 全域扫描）。
//
// 不手写清单：flow 新增状态时本用例自动覆盖到它，逼着对账映射被有意识地归过类，
// 而不是默默落到 default（不对账）——那在生产里的表现就是提案永久停在 pending
func allProcinstStatuses() []flowentity.ProcinstStatus {
	var all []flowentity.ProcinstStatus
	for v := -128; v < 128; v++ {
		if s := flowentity.ProcinstStatus(v); flowentity.ProcinstStatusEnum.Valid(s) == nil {
			all = append(all, s)
		}
	}
	return all
}

// TestReconciledProposalStatusCoversEveryProcinstStatus 工单状态 → 提案对账状态的全映射。
//
// 穷尽性由枚举反查保证：既拦住「新状态未归类」，也拦住「用例登记了已不存在的状态」
func TestReconciledProposalStatusCoversEveryProcinstStatus(t *testing.T) {
	want := map[flowentity.ProcinstStatus]string{
		flowentity.ProcinstStatusCancelled:  entity.ProposalStatusRejected,
		flowentity.ProcinstStatusTerminated: entity.ProposalStatusRejected,
		flowentity.ProcinstStatusCompleted:  entity.ProposalStatusApproved,
		flowentity.ProcinstStatusActive:     "",
		flowentity.ProcinstStatusSuspended:  "",
		flowentity.ProcinstStatusBack:       "",
	}

	all := allProcinstStatuses()
	require.NotEmpty(t, all, "枚举反查失效：一个工单状态也没扫到，本用例已失去穷尽性")
	for _, s := range all {
		desc := flowentity.ProcinstStatusEnum.GetDesc(s)
		expected, known := want[s]
		require.True(t, known, "工单状态 %d(%s) 未被本用例登记，对账映射必须对每个状态有意识归类", s, desc)
		require.Equal(t, expected, reconciledProposalStatus(s), "工单状态 %s 的对账结果不符", desc)
	}
	require.Len(t, want, len(all), "用例登记了枚举里不存在的工单状态，说明清单已过期")
}

// --- Delete：仅终态可删 ---

func TestProposalDeleteTerminalOnly(t *testing.T) {
	gw, _ := setupProposalGateway(t)
	stubFlow(t, nil, nil)

	pending := &entity.Proposal{Action: "a", Status: entity.ProposalStatusPending, OwnerAccountId: 1}
	require.NoError(t, gw.Save(context.Background(), pending))
	require.Error(t, gw.Delete(context.Background(), pending.Id), "非终态不可删（避免删掉在途审批）")

	executed := &entity.Proposal{Action: "a", Status: entity.ProposalStatusExecuted, OwnerAccountId: 1}
	require.NoError(t, gw.Save(context.Background(), executed))
	require.NoError(t, gw.Delete(context.Background(), executed.Id), "终态可删")

	require.Error(t, gw.Delete(context.Background(), 999999), "不存在的提案删除报错")
}

// --- SweepStalledProposals：进程内队列丢单后的自愈 ---

// 一组能通过 action 准入与参数形状校验的锁定参数：停滞重投必须能真的把执行跑起来，
// 否则用例测不到「重投」而只测到「改状态」
const (
	stalledProposalAction = "MachineCommandExec"
	stalledProposalParams = `{"authCertName":"c1","command":"true","remark":"停滞重投用例"}`
)

// backdateProposal 把提案的 update_time 推到过去，模拟「领取后进程就退出了」。
//
// 停滞判定读的就是 update_time（approved 是领取那刻写入的）。回读断言必须带着：
// 若回写被基类钩子刷新成当前时间，用例会以「一条也没扫到」的方式假绿。
// 参数取仓储而不是网关：治理看板的停滞计数与清扫共用同一判据，两边用例都要造这个形态
func backdateProposal(t *testing.T, repo repository.Proposal, id uint64, age time.Duration) {
	t.Helper()
	old := time.Now().Add(-age)
	_, err := repo.UpdateColsByCond(context.Background(),
		base.NewPatch().Set("updateTime", old),
		model.NewCond().EqStrict("id", id))
	require.NoError(t, err)

	got, gerr := repo.GetById(id)
	require.NoError(t, gerr)
	require.NotNil(t, got.UpdateTime)
	require.WithinDuration(t, old, *got.UpdateTime, time.Minute, "update_time 必须真的被推到过去，否则停滞判定根本测不到")
}

// TestSweepStalledProposalsRequeuesLostExecution 停滞清扫必须真的把丢失的执行重新推进到终态，
// 且只动「approved 且无执行运行且已过静默期」那一类。
//
// 真机形态：受控执行投递的是进程内队列，进程在 worker 取出前退出就丢了单；而工单在回调
// 返回「已受理」那一刻已被 flow 记为处理成功，于是 flow 的清扫/改期/取消执行全部不再可用，
// FlowBizHandle 也不会再被调用——没有这条清扫，提案就永久停在 approved
func TestSweepStalledProposalsRequeuesLostExecution(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	stubFlow(t, nil, nil)
	var execCalls int32
	fakeRunner(t, func(_ context.Context, req *headless.RunReq) (*headless.RunResult, error) {
		if req.ProfileId == headless.ProfileExecutor {
			atomic.AddInt32(&execCalls, 1)
		}
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: "done"}, nil
	})

	staleAfter := proposalStallAfter + time.Hour
	seed := func(status string, execRunId uint64, backdate bool) *entity.Proposal {
		p := &entity.Proposal{
			Action: stalledProposalAction, Params: stalledProposalParams, Risk: "low", Reason: "r",
			Status: status, OwnerAccountId: 1, ExecRunId: execRunId,
		}
		require.NoError(t, gw.Save(context.Background(), p))
		if backdate {
			backdateProposal(t, gw.GetRepo(), p.Id, staleAfter)
		}
		return p
	}

	stalled := seed(entity.ProposalStatusApproved, 0, true)    // 丢单：必须被重投
	fresh := seed(entity.ProposalStatusApproved, 0, false)     // 刚领取：可能正在跑，不得动
	withRun := seed(entity.ProposalStatusApproved, 4242, true) // 已有执行运行：不属停滞
	done := seed(entity.ProposalStatusExecuted, 4243, true)    // 终态：不得重做
	unapproved := seed(entity.ProposalStatusPending, 0, true)  // 未审批：绝不能因停滞而被执行

	requeued, err := gw.SweepStalledProposals(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, requeued, "只有「approved 且无执行运行且已过静默期」这一条算停滞")

	waitProposalStatus(t, gw, stalled.Id, entity.ProposalStatusExecuted)
	require.Equal(t, int32(1), atomic.LoadInt32(&execCalls), "重投必须真的跑一次执行档，而不是只改状态")

	landed, gerr := gw.GetById(stalled.Id)
	require.NoError(t, gerr)
	require.NotZero(t, landed.ExecRunId, "重投执行后必须写下执行运行关联，否则下一轮清扫会无限重投")

	// 逐条核对清扫不该碰的提案：状态与执行运行关联都必须原样保留。
	// 关联值放进表结构而不是统一断言为零——「已有执行运行」与「已执行」本就带着关联，
	// 清扫若把它们抹成 0，下一轮就会把已落地的变更再执行一遍
	for _, untouched := range []struct {
		name          string
		id            uint64
		wantStatus    string
		wantExecRunId uint64
	}{
		{"刚领取的在途提案", fresh.Id, entity.ProposalStatusApproved, 0},
		{"已有执行运行的提案", withRun.Id, entity.ProposalStatusApproved, 4242},
		{"已执行的终态提案", done.Id, entity.ProposalStatusExecuted, 4243},
		{"尚未审批的提案", unapproved.Id, entity.ProposalStatusPending, 0},
	} {
		got, err := gw.GetById(untouched.id)
		require.NoError(t, err)
		require.Equal(t, untouched.wantStatus, got.Status, "%s 不得被停滞清扫改动", untouched.name)
		require.Equal(t, untouched.wantExecRunId, got.ExecRunId, "%s 的执行运行关联不得被改写", untouched.name)
	}

	// 只应产出停滞那一条的执行档 + 校验档，其余提案一个运行也不得派生
	cnt, cerr := runSvc.CountByCond(&entity.Run{})
	require.NoError(t, cerr)
	require.Equal(t, int64(2), cnt, "重投一条提案 = 执行档 + 校验档共两个运行，不多不少")
}

// TestExecuteApprovedLeavesInFlightExecRunAlone 幂等键短路撞上在途运行时不得判败。
//
// RunSync 命中幂等键会原样返回既有运行，其状态可能仍是 running（多实例部署下另一实例正在执行，
// 或停滞重投时上一任执行者还没退出）。把它当失败会把一条正在执行的提案标成 failed，
// 而真实执行稍后成功落地——「提案说失败、变更其实生效」，且 failed 是终态再无人回头看，
// 工单还会被连带改判成处理失败
func TestExecuteApprovedLeavesInFlightExecRunAlone(t *testing.T) {
	gw, runSvc := setupProposalGateway(t)
	inst := &fakeProcinst{}
	stubFlow(t, nil, inst)
	var calls int32
	fakeRunner(t, func(_ context.Context, _ *headless.RunReq) (*headless.RunResult, error) {
		atomic.AddInt32(&calls, 1)
		return &headless.RunResult{Status: headless.RunStatusSucceeded, Output: "done"}, nil
	})

	p := &entity.Proposal{
		Action: stalledProposalAction, Params: stalledProposalParams, Risk: "low", Reason: "r",
		Status: entity.ProposalStatusApproved, OwnerAccountId: 1, ProcInstId: 9400,
	}
	require.NoError(t, gw.Save(context.Background(), p))
	pidStr := strconv.FormatUint(p.Id, 10)

	// 预置一条同幂等键、仍在途的执行运行：正是「另一实例正在执行这条提案」的库内形态
	inFlight := &entity.Run{
		RunNo: "rn-inflight", TriggerType: entity.TriggerTypeManual, TriggerRef: "proposal:" + pidStr,
		ProfileId: headless.ProfileExecutor, Title: p.Title, Status: entity.RunStatusRunning,
		OwnerAccountId: 1, IdempotencyKey: "proposal-exec:" + pidStr, Input: "已批准变更",
	}
	require.NoError(t, runSvc.GetRepo().Insert(context.Background(), inFlight))

	gw.processApproved(p.Id)

	saved, err := gw.GetById(p.Id)
	require.NoError(t, err)
	require.Equal(t, entity.ProposalStatusApproved, saved.Status, "在途执行不是失败：必须把提案留给它的归属者收尾")
	require.Empty(t, saved.Err, "不得写下失败原因")
	require.Zero(t, saved.ExecRunId, "在途运行的关联由其归属者回写，本次不得抢写")
	require.Equal(t, int32(0), atomic.LoadInt32(&calls), "命中幂等键必须短路，一次 LLM 也不该再烧")
	require.Empty(t, inst.bizRecs, "未到终态就不得回写工单，否则工单会比提案先宣布结果")

	cnt, cerr := runSvc.CountByCond(&entity.Run{})
	require.NoError(t, cerr)
	require.Equal(t, int64(1), cnt, "不得新建第二条执行运行（那才是把变更做两遍）")
}
