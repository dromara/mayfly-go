package application

import (
	"context"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/imsg"
	aitools "mayfly-go/internal/ai/tools"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

const (
	// proposalExecWorkers 受控执行串行度（审批后的变更一次跑一个，防并发互踩同一资源）
	proposalExecWorkers = 1
	// proposalExecQueueSize 受控执行队列容量（满则对审批请求形成背压，不静默丢单）
	proposalExecQueueSize = 128
	// proposalStallSweepBatch 单轮停滞清扫的处理上限（控制单轮工作量，不是安全边界）。
	//
	// 真正的安全边界在投递方式：清扫走 non-blocking 的 trySubmitForExecution，队列满就停下；
	// 若拿阻塞式的 submitForExecution 去做清扫（它带背压语义、等空位且不报错），在积压超过队列容量时
	// 就会把调用方拖住——Init 里那轮清扫排在 Run 工作池启动与 cron 绑定之前，被拖住的表现是
	// 「进程已起、HTTP 可用，但队列不消费、定时任务不调度」
	proposalStallSweepBatch = 100

	// proposalStallAfter 判定「已领取执行权但受控执行从未落库」所需的静默期。
	//
	// 取默认租约上界之上：执行档 + 校验档各为 defaultRunTimeout + runSweepGraceSeconds（≈14min），
	// 故默认配置下正常在途的执行不会被误判为停滞。它只是降噪阈值而**不是正确性边界**——重投的
	// 安全性由执行运行的幂等键（proposal-exec:<id>）与 executeApproved 的在途识别共同保证，
	// 因此任务自定义了更长超时也只是多等几轮清扫，不会把变更执行第二遍
	proposalStallAfter = 15 * time.Minute

	// proposalStallSweepInterval 停滞提案清扫周期（与运行租约清扫同量级）
	proposalStallSweepInterval = 5 * time.Minute
)

// AgentRemediationFlowBizType 变更提案审批的流程业务类型（复刻 db 的 DbSQLExecFlowBizType 范式）。
//
// 提案创建 → 以此 BizType 发起 flow 审批工单；agentops init 期经 flowapp.RegisterBizHandler
// 注册本网关为回调处理器，审批通过后 flow 回调 FlowBizHandle 触发受控执行。
const AgentRemediationFlowBizType = "agent_remediation_flow"

// defaultRemediationProcDefKey 变更提案审批流程定义的默认 DefKey。
//
// 审批流定义（procdef）由管理员在流程设计器中预先创建并启用（对齐 flow 既有「管理员配置审批流」
// 模型）；本网关按此 DefKey 定位生效的流程定义。可用 SetRemediationProcDefKey 覆盖。
const defaultRemediationProcDefKey = "agent_remediation"

// bizKeyPrefix 审批工单 BizKey 前缀（BizKey=aiops-proposal:<id>，便于工单侧反查提案）
const bizKeyPrefix = "aiops-proposal:"

var remediationProcDefKey = defaultRemediationProcDefKey

// SetRemediationProcDefKey 覆盖变更提案审批流程定义的 DefKey（须在首次提案登记前调用）
func SetRemediationProcDefKey(key string) {
	if strings.TrimSpace(key) != "" {
		remediationProcDefKey = strings.TrimSpace(key)
	}
}

// flow 依赖的窄接口子集：网关仅用到这些方法。窄化（而非直接依赖庞大的 flowapp.Procdef/Procinst，
// 二者内嵌 base.App 约 25 个方法）使单测可注入最小 fake，无需装配整个 flow 模块。
type (
	// procdefFinder 按条件定位流程定义（网关据 DefKey 查审批流定义）
	procdefFinder interface {
		GetByCond(cond any) error
	}
	// procinstRunner 发起审批工单 + 按 id 读工单（惰性对账读终态）+ 终态回写处理结果。
	// 回写通道必需：受控执行是异步的，回调只能给「已受理」回执，不把真实结果写回工单列，
	// 审批人回到工单永远只看到回执（真实结果在提案侧，但工单候选人不必有提案页可见性）
	procinstRunner interface {
		StartProc(ctx context.Context, procdefId uint64, reqParam *flowdto.StarProc) (*flowentity.Procinst, error)
		GetById(id uint64, cols ...string) (*flowentity.Procinst, error)
		RecordBizHandleResult(ctx context.Context, procinstId uint64, bizStatus flowentity.ProcinstBizStatus, handleRes string) error
	}
)

// flow 应用解析间接层：生产经 ioc 取 flow 应用（flow 已在启动期注册，故非 nil）；单测覆写这两个
// 变量为返回最小 fake / nil 的函数，即可在不装配 flow 模块的前提下覆盖提案登记、审批发起与状态对账。
var (
	flowProcdefFn  = func() procdefFinder { return flowapp.GetProcdefApp() }
	flowProcinstFn = func() procinstRunner { return flowapp.GetProcinstApp() }
)

// ProposalGateway 变更提案网关（自主运维闭环的「提案 → 审批 → 受控执行 → 验证」中枢）。
//
// 端口-适配器：
//   - 实现 ai/tools.ProposalRecorder：propose_remediation 工具经此登记提案 + 发起 flow 审批工单
//     （ai 面向端口编程，agentops 注入实现，破除 ai → agentops 反向依赖）。
//   - 实现 flow.FlowBizHandler：审批通过回调 → 执行档按锁定参数受控执行 → 只读校验档复查 →
//     结论回写提案终态（完全对齐 db 的 DbSQLExecFlowBizType BizHandler 范式）。
type ProposalGateway interface {
	base.App[*entity.Proposal]

	// GetPageList 分页查询提案
	GetPageList(condition *entity.Proposal, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Proposal], error)

	// Delete 删除提案（仅终态可删，避免删掉在途审批）
	Delete(ctx context.Context, id uint64) error

	// Record 登记变更提案（实现 ai/tools.ProposalRecorder）：落库 + 视配置发起 flow 审批工单
	Record(ctx context.Context, draft *aitools.ProposalDraft) (*aitools.ProposalRecord, error)

	// FlowBizHandle 审批通过回调（实现 flow.FlowBizHandler）：领取执行权后投递给异步执行体，立即返回
	FlowBizHandle(ctx context.Context, param *flowapp.BizHandleParam) (any, error)

	// StartExecutionWorker 启动受控执行异步体（应用 Init 期调用，幂等）。
	// 必须先于工作池启动：审批回调可能在任何时刻到来
	StartExecutionWorker()

	// StopExecutionWorker 优雅停止受控执行异步体（等待在途提案收尾）
	StopExecutionWorker()

	// SyncProcStatus 惰性对账：按审批工单的终态把仍停在「待审批」的提案校准到应有状态
	//（取消/终止 → rejected，完成 → approved）。flowEventBus 为 flow 内部总线，agentops 无法订阅，
	// 故在读取路径 best-effort 对账。只改状态、绝不触发执行（读路径不得产生下发命令这类副作用）
	SyncProcStatus(ctx context.Context, proposals []*entity.Proposal)

	// SweepStalledProposals 重投「已领取执行权但受控执行从未落库」的停滞提案，返回本轮重投条数。
	// 应用 Init 期调一次，并由受控执行体内的周期清扫重跑（进程内队列丢单后唯一的自愈路径）
	SweepStalledProposals(ctx context.Context) (int, error)
}

type proposalGatewayImpl struct {
	base.AppImpl[*entity.Proposal, repository.Proposal]

	// runService 运行聚合服务：审批通过后的执行档/校验档运行经其 RunSync 同步产出 Run
	// （在受控执行异步体的 goroutine 里同步跑，不在审批请求链路上）
	runService RunService `inject:"T"`

	// 受控执行异步体：队列 + worker（由 StartExecutionWorker 拉起）
	execQueue   chan uint64
	execStop    chan struct{}
	execWg      sync.WaitGroup
	execMu      sync.Mutex
	execStarted bool
}

var _ ProposalGateway = (*proposalGatewayImpl)(nil)
var _ aitools.ProposalRecorder = (*proposalGatewayImpl)(nil)
var _ flowapp.FlowBizHandler = (*proposalGatewayImpl)(nil)

// registerProposalIntegration 把提案网关接入 flow 审批回调与 ai 提案登记端口（init 期一次）。
//
// 依赖倒置：ai/tools 面向 ProposalRecorder 端口编程，此处注入 agentops 实现，破除 ai → agentops
// 反向依赖；flow 经 RegisterBizHandler 绑定 AgentRemediationFlowBizType，审批通过后回调本网关执行。
// 必须在 RunService.Start() 之前调用：各 initFunc 由 starter 以独立 goroutine 并发启动，而工作池一旦
// 认领到调用 propose_remediation 的运行，登记端口须已就绪（否则该次提案会走 nil-recorder 降级）。
func registerProposalIntegration() {
	gateway := GetProposalGateway()
	gateway.StartExecutionWorker()
	flowapp.RegisterBizHandler(AgentRemediationFlowBizType, gateway)
	aitools.SetProposalRecorder(gateway)
	logx.Infof("[agentops] proposal integration registered: flowBizType=%s, procDefKey=%s",
		AgentRemediationFlowBizType, remediationProcDefKey)
}

// AgentRemediationBizForm 审批工单业务表单（携带提案详情，供审批人判断）
type AgentRemediationBizForm struct {
	ProposalId     uint64 `json:"proposalId"`
	Title          string `json:"title"`
	Action         string `json:"action"`
	Params         string `json:"params"`
	Risk           string `json:"risk"`
	Reason         string `json:"reason"`
	RunId          uint64 `json:"runId"`
	ConversationId uint64 `json:"conversationId"`
	OwnerUsername  string `json:"ownerUsername"`
}

// ticketHandleResult 受控执行终态回写到审批工单的处理结果（biz_handle_res 列）。
//
// 只存摘要文本与运行 id，不存全文：提案行才是执行/校验结果的单一真源，这里是为审批人
// 可读性的有界投影（列宽由截断保证，运行 id 供去运行详情看全文）
type ticketHandleResult struct {
	ProposalId uint64 `json:"proposalId"`
	// Status 提案终态（executed/failed），前端据此区分「真实结果」与旧版「已受理」回执
	Status           string `json:"status"`
	ExecRunId        uint64 `json:"execRunId,omitempty"`
	ExecResult       string `json:"execResult,omitempty"`
	VerifyRunId      uint64 `json:"verifyRunId,omitempty"`
	VerifyConclusion string `json:"verifyConclusion,omitempty"`
	Err              string `json:"err,omitempty"`
}

// maxTicketResultLen 结果摘要在工单列里的单字段截断长度（text 列容量充裕，这里只为防模型超长输出撞大行）
const maxTicketResultLen = 4000

// truncateTicketText 按 rune 截断，不劈多字节字符
func truncateTicketText(s string) string {
	rs := []rune(s)
	if len(rs) <= maxTicketResultLen {
		return s
	}
	return string(rs[:maxTicketResultLen]) + "…"
}

// recordTicketResult 把提案终态回写到审批工单（best-effort：失败只记日志，提案行事实不受影响）。
//
// 只在终态（executed/failed）调用：approved 中间态回写会把回执换成无结论的半成品，而工单回执
// 本就表达「已受理」；执行失败时连带把工单 biz_status 从「处理成功」改判为「处理失败」，
// 否则工单列表说批完一切正常而变更根本没落地
func (g *proposalGatewayImpl) GetPageList(condition *entity.Proposal, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Proposal], error) {
	return g.GetRepo().GetPageList(condition, pageParam, orderBy...)
}

// Delete 删除提案（仅终态可删）
func (g *proposalGatewayImpl) Delete(ctx context.Context, id uint64) error {
	proposal, err := g.GetById(id)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrProposalNotFound)
	}
	if !entity.IsProposalTerminal(proposal.Status) {
		return errorx.NewBizI(ctx, imsg.ErrProposalNotTerminal)
	}
	return g.DeleteById(ctx, id)
}

// Record 登记变更提案：落库 pending → 解析审批流定义 → 发起 flow 审批工单 → 回写工单关联。
//
// fail-soft：提案一旦落库即视为登记成功；未绑定审批流 / 发起工单失败时不返回错误（提案仍在，
// 处于待审批），而是把情况写进面向模型的 Message，让 Agent 知晓并据此产出只读结论。
func (g *proposalGatewayImpl) Record(ctx context.Context, draft *aitools.ProposalDraft) (*aitools.ProposalRecord, error) {
	if draft == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrProposalDraftNil)
	}
	if strings.TrimSpace(draft.Action) == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrProposalActionRequired)
	}
	// action 必须是「可执行的变更」：在登记时就拦下，而不是等审批通过后才发生「批了却执行不了」。
	// 与 allowedToolsForAction 共用同一判据，两侧不会强度漂移
	if err := validateProposalAction(ctx, strings.TrimSpace(draft.Action), draft.Params); err != nil {
		return nil, err
	}

	proposal := &entity.Proposal{
		RunId:          draft.RunId,
		ConversationId: draft.ConversationId,
		Title:          strings.TrimSpace(draft.Title),
		Action:         strings.TrimSpace(draft.Action),
		Params:         draft.Params,
		Risk:           draft.Risk,
		Reason:         draft.Reason,
		Status:         entity.ProposalStatusPending,
		OwnerAccountId: draft.OwnerAccountId,
		OwnerUsername:  draft.OwnerUsername,
	}
	if proposal.Title == "" {
		proposal.Title = proposal.Action
	}
	if proposal.Risk == "" {
		proposal.Risk = entity.ProposalRiskHigh
	}
	// 归属身份兜底：草稿未带则取运行上下文登录账号（headless 运行已注入 owner）
	if proposal.OwnerAccountId == 0 {
		if la := contextx.GetLoginAccount(ctx); la != nil {
			proposal.OwnerAccountId = la.Id
			proposal.OwnerUsername = la.Username
		}
	}
	// fail-closed：无归属身份拒绝登记（审批发起人/执行身份均依赖 owner）
	if proposal.OwnerAccountId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrProposalOwnerRequired)
	}

	if err := g.Save(ctx, proposal); err != nil {
		return nil, err
	}
	logx.InfofContext(ctx, "[agentops-proposal] recorded: id=%d, action=%s, risk=%s, runId=%d",
		proposal.Id, proposal.Action, proposal.Risk, proposal.RunId)

	// 解析审批流定义：未绑定则提案停留在待审批（提示管理员配置），不阻断登记
	def := g.resolveProcDef(ctx)
	if def == nil {
		logx.WarnfContext(ctx, "[agentops-proposal] no enabled procdef bound (defKey=%s); proposal[%d] stays pending",
			remediationProcDefKey, proposal.Id)
		return &aitools.ProposalRecord{
			ProposalId: proposal.Id,
			Status:     proposal.Status,
			Message: "变更提案已登记（待审批），但系统尚未绑定审批流程（流程定义 DefKey=" + remediationProcDefKey +
				"）。请联系管理员在流程管理中创建并启用该审批流后，提案方可进入审批与受控执行。当前请勿尝试直接执行变更。",
		}, nil
	}

	procinst, err := g.startApprovalProc(ctx, def, proposal)
	if err != nil {
		// 提案已落库，发起工单失败不回滚提案（fail-soft）：明确告知模型，避免其误以为已提交审批
		logx.ErrorfContext(ctx, "[agentops-proposal] start approval proc failed: proposalId=%d, err=%v", proposal.Id, err)
		return &aitools.ProposalRecord{
			ProposalId: proposal.Id,
			Status:     proposal.Status,
			Message:    "变更提案已登记（待审批），但发起审批工单失败：" + err.Error(),
		}, nil
	}

	// 回写工单关联（审批回调/状态对账据此定位提案）
	if _, uerr := g.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().Set("procInstId", procinst.Id).Set("procDefId", def.Id),
		model.NewCond().EqStrict("id", proposal.Id)); uerr != nil {
		logx.ErrorfContext(ctx, "[agentops-proposal] bind procinst failed: proposalId=%d, procInstId=%d, err=%v",
			proposal.Id, procinst.Id, uerr)
	}

	return &aitools.ProposalRecord{
		ProposalId: proposal.Id,
		ProcInstId: procinst.Id,
		Status:     proposal.Status,
		Message:    "变更提案已登记并发起审批工单，等待人工审批；审批通过后系统将按锁定参数自动受控执行并做只读校验。请勿自行执行该变更。",
	}, nil
}
