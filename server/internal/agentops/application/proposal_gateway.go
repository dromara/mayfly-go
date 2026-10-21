package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	aitools "mayfly-go/internal/ai/tools"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
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
func (g *proposalGatewayImpl) recordTicketResult(ctx context.Context, proposalId uint64) {
	proposal, err := g.GetById(proposalId)
	if err != nil || proposal == nil || proposal.ProcInstId == 0 {
		return
	}
	bizStatus := flowentity.ProcinstBizStatusSuccess
	if proposal.Status != entity.ProposalStatusExecuted {
		bizStatus = flowentity.ProcinstBizStatusFail
	}
	app := flowProcinstFn()
	if app == nil {
		// flow 应用未就绪（启动早期/单测未装配）：丢一次回写不影响提案自身事实
		return
	}
	res := &ticketHandleResult{
		ProposalId:       proposal.Id,
		Status:           proposal.Status,
		ExecRunId:        proposal.ExecRunId,
		ExecResult:       truncateTicketText(proposal.ExecResult),
		VerifyRunId:      proposal.VerifyRunId,
		VerifyConclusion: truncateTicketText(proposal.VerifyConclusion),
		Err:              proposal.Err,
	}
	if uerr := app.RecordBizHandleResult(ctx, proposal.ProcInstId, bizStatus, jsonx.ToStr(res)); uerr != nil {
		logx.WarnfContext(ctx, "[agentops-proposal] record ticket result failed (non-fatal): proposalId=%d, procInstId=%d, err=%v",
			proposal.Id, proposal.ProcInstId, uerr)
	}
}

// GetPageList 分页查询提案
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

// resolveProcDef 按 DefKey 定位生效的审批流程定义（不存在/未启用返回 nil）
func (g *proposalGatewayImpl) resolveProcDef(ctx context.Context) *flowentity.Procdef {
	procdefApp := flowProcdefFn()
	if procdefApp == nil {
		return nil
	}
	def := &flowentity.Procdef{DefKey: remediationProcDefKey}
	if err := procdefApp.GetByCond(def); err != nil {
		if !model.IsNotFound(err) {
			logx.WarnfContext(ctx, "[agentops-proposal] resolve procdef failed: defKey=%s, err=%v", remediationProcDefKey, err)
		}
		return nil
	}
	if def.Status != flowentity.ProcdefStatusEnable {
		return nil
	}
	return def
}

// startApprovalProc 以提案详情为业务表单发起 flow 审批工单（BizType=AgentRemediationFlowBizType）
func (g *proposalGatewayImpl) startApprovalProc(ctx context.Context, def *flowentity.Procdef, proposal *entity.Proposal) (*flowentity.Procinst, error) {
	procinstApp := flowProcinstFn()
	if procinstApp == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrProposalFlowNotReady)
	}
	bizForm := &AgentRemediationBizForm{
		ProposalId:     proposal.Id,
		Title:          proposal.Title,
		Action:         proposal.Action,
		Params:         proposal.Params,
		Risk:           proposal.Risk,
		Reason:         proposal.Reason,
		RunId:          proposal.RunId,
		ConversationId: proposal.ConversationId,
		OwnerUsername:  proposal.OwnerUsername,
	}
	return procinstApp.StartProc(ctx, def.Id, &flowdto.StarProc{
		BizType: AgentRemediationFlowBizType,
		BizKey:  bizKeyPrefix + strconv.FormatUint(proposal.Id, 10),
		BizForm: jsonx.ToStr(bizForm),
		Remark:  "[AI变更提案] " + proposal.Title,
	})
}

// FlowBizHandle 审批通过回调：领取执行权 → 投递受控执行异步体 → 立即返回。
//
// 仅在流程「已完成」（审批通过）时触发执行；驳回/取消不经此回调（由 SyncProcStatus 惰性对账）。
// 幂等：以 pending→approved 的 CAS 领取执行权，flow 清扫重驱动/并发回调时只有一方真正领取。
//
// 为什么不在这里直接跑执行：本回调由审批请求事务提交后的动作在同一 goroutine 内同步执行（也曾被
// 清扫任务串行重驱动）。里面直接跑「执行档 + 校验档」两次无头运行（默认各 5min）会把审批接口挂到
// 网关超时——而 SQL/命令其实已经执行（不可回滚），审批人看到「请求失败」极易再操作一次；同时
// RunSync 绕过有界工作池，本特性唯一的并发上限在这里会失效。
func (g *proposalGatewayImpl) FlowBizHandle(ctx context.Context, param *flowapp.BizHandleParam) (any, error) {
	if param == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrProposalBizParamNil)
	}
	procinst := param.Procinst
	if procinst.Status != flowentity.ProcinstStatusCompleted {
		// 非完成状态不处理（与 db 的 FlowBizHandle 口径一致）
		return nil, nil
	}

	proposal, err := g.locateByProcInst(ctx, procinst)
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		// 两种途径都定位不到（非本模块工单或提案已删）：忽略，不报错影响 flow
		return nil, nil
	}
	if entity.IsProposalTerminal(proposal.Status) {
		return map[string]any{AckKeyProposalId: proposal.Id, AckKeyStatus: proposal.Status, AckKeyMessage: AckMessageAlreadyProcessed}, nil
	}

	// CAS 领取执行权：仅 pending → approved；落败说明已被其它路径领取（并发/重驱动）
	rows, err := g.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().Set("status", entity.ProposalStatusApproved),
		model.NewCond().EqStrict("id", proposal.Id).EqStrict("status", entity.ProposalStatusPending))
	if err != nil {
		return nil, err
	}
	if rows != 1 {
		return g.handleClaimLost(ctx, proposal.Id)
	}

	if err := g.submitForExecution(ctx, proposal.Id); err != nil {
		return nil, err
	}
	logx.InfofContext(ctx, "[agentops-proposal] approved, execution queued: proposalId=%d, action=%s", proposal.Id, proposal.Action)
	return map[string]any{
		AckKeyProposalId: proposal.Id,
		AckKeyStatus:     entity.ProposalStatusApproved,
		AckKeyExecution:  AckExecutionQueued,
	}, nil
}

// handleClaimLost CAS 落败后的收敛：
//   - 已终态或非本次能接的状态：幂等返回，不重复执行
//   - 仍停在 approved 且 exec_run_id=0：说明上一次领取后崩在执行之前 → 重新投递执行。
//     执行运行的幂等键 proposal-exec:<id> 保证「已真正执行过」时不会把变更再做第二遍
//     （命中幂等键短路返回既有运行）。
//     注意这条兜底只在 FlowBizHandle 被再次调用时才生效，而 flow 只重驱动「业务待处理」的工单，
//     本模块的工单在回调返回那一刻就已记为处理成功——所以它拦的是并发竞态，
//     真正的丢单自愈靠 SweepStalledProposals
func (g *proposalGatewayImpl) handleClaimLost(ctx context.Context, proposalId uint64) (any, error) {
	reloaded, err := g.GetById(proposalId)
	if err != nil || reloaded == nil {
		return map[string]any{AckKeyProposalId: proposalId, AckKeyMessage: AckMessageAlreadyClaimed}, nil
	}
	if reloaded.Status == entity.ProposalStatusApproved && reloaded.ExecRunId == 0 {
		if err := g.submitForExecution(ctx, reloaded.Id); err != nil {
			return nil, err
		}
		logx.WarnfContext(ctx, "[agentops-proposal] requeue orphaned approved proposal (previous executor crashed before start): proposalId=%d", proposalId)
		return map[string]any{AckKeyProposalId: proposalId, AckKeyStatus: reloaded.Status, AckKeyExecution: AckExecutionRequeued}, nil
	}
	return map[string]any{AckKeyProposalId: proposalId, AckKeyStatus: reloaded.Status, AckKeyMessage: AckMessageAlreadyClaimed}, nil
}

// locateByProcInst 按工单定位提案：先走 proc_inst_id；关联未落库（Record 回写失败）时按工单
// 业务键 aiops-proposal:<id> 反查兜底，并补回关联。
//
// 为什么必须有兜底：发起工单与回写 procInstId 是两次写（提案落库即视为登记成功），回写失败时若只按
// procInstId 找，工单批完也找不到提案 → 提案永久 pending、工单显示「已完成」、模型收到「已发起审批」
// 的回复，三处都说没问题而变更永不执行，是最难发现的一类失效。
// BizKey 与工单在同一次发起里生成（不依赖那次回写），所以它能作为可靠的反向定位键。
func (g *proposalGatewayImpl) locateByProcInst(ctx context.Context, procinst flowentity.Procinst) (*entity.Proposal, error) {
	proposal, err := g.GetRepo().GetByProcInstId(procinst.Id)
	if err == nil {
		return proposal, nil
	}
	if !model.IsNotFound(err) {
		return nil, err
	}
	id, ok := proposalIdFromBizKey(procinst.BizKey)
	if !ok {
		return nil, nil
	}
	repaired, getErr := g.GetById(id)
	if getErr != nil || repaired == nil {
		return nil, nil
	}
	// 补回关联（仅当仍未绑定，避免覆盖并发写入的其它工单关联）
	if _, uerr := g.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().Set("procInstId", procinst.Id).Set("procDefId", procinst.ProcdefId),
		model.NewCond().EqStrict("id", repaired.Id).EqStrict("procInstId", 0)); uerr != nil {
		logx.ErrorfContext(ctx, "[agentops-proposal] repair procinst binding failed (non-fatal): proposalId=%d, err=%v", repaired.Id, uerr)
	}
	logx.WarnfContext(ctx, "[agentops-proposal] proposal located by biz key (proc_inst_id was not bound): proposalId=%d, procInstId=%d", repaired.Id, procinst.Id)
	repaired.ProcInstId = procinst.Id
	return repaired, nil
}

// proposalIdFromBizKey 从工单业务键解析提案 id（非本模块键返回 false）
func proposalIdFromBizKey(bizKey string) (uint64, bool) {
	if !strings.HasPrefix(bizKey, bizKeyPrefix) {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(bizKey, bizKeyPrefix), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

// StartExecutionWorker 启动受控执行异步体（幂等）
func (g *proposalGatewayImpl) StartExecutionWorker() {
	g.execMu.Lock()
	defer g.execMu.Unlock()
	if g.execStarted {
		return
	}
	g.execStarted = true
	g.execQueue = make(chan uint64, proposalExecQueueSize)
	g.execStop = make(chan struct{})
	for i := 0; i < proposalExecWorkers; i++ {
		g.execWg.Add(1)
		go g.execWorker()
	}
	// 停滞清扫与执行体同生命周期：重投依赖「投递能力」，执行体停止后再投只会撞上未启动的队列
	g.execWg.Add(1)
	go g.stallSweeper()
	logx.Infof("[agentops-proposal] controlled-execution worker started: workers=%d, queueCapacity=%d, stallSweepInterval=%v",
		proposalExecWorkers, proposalExecQueueSize, proposalStallSweepInterval)
}

// StopExecutionWorker 优雅停止受控执行异步体（等待在途提案收尾）
func (g *proposalGatewayImpl) StopExecutionWorker() {
	g.execMu.Lock()
	if !g.execStarted {
		g.execMu.Unlock()
		return
	}
	g.execStarted = false
	g.execMu.Unlock()

	close(g.execStop)
	g.execWg.Wait()
	logx.Infof("[agentops-proposal] controlled-execution worker stopped")
}

// trySubmitForExecution 非阻塞投递（仅供停滞清扫）。
//
// 为什么不能用 submitForExecution：审批回调那条路径有调用方，队列满时阻塞等待是**背压**；
// 而清扫路径没有任何调用方可以背压，等下去只会把调用者（尤其 Init，它在工作池启动与 cron
// 绑定之前）拖住，且这种阻塞不产生错误、日志里只有一行 Warn——实测被自己的回归用例逮到。
// 返回 false 表示「本轮没投进去」，留到下一轮清扫（幂等键保证重投不重复执行）
func (g *proposalGatewayImpl) trySubmitForExecution(proposalId uint64) bool {
	g.execMu.Lock()
	queue, stop, started := g.execQueue, g.execStop, g.execStarted
	g.execMu.Unlock()
	if !started {
		return false
	}
	// 先非阻塞确认未停机：select 在多个就绪分支间随机选择，若与 send 放在一起，
	// 停机后仍可能把 id 塞进没人消费的缓冲而假装投递成功
	select {
	case <-stop:
		return false
	default:
	}
	select {
	case queue <- proposalId:
		return true
	default:
		return false // 队列满：本轮到此为止
	}
}

// submitForExecution 把提案投递给异步执行体：队列满时形成背压（等空位或调用方 ctx 结束），
// 不静默丢单——丢单等于「批过的变更没人执行」
func (g *proposalGatewayImpl) submitForExecution(ctx context.Context, proposalId uint64) error {
	g.execMu.Lock()
	queue, stop, started := g.execQueue, g.execStop, g.execStarted
	g.execMu.Unlock()
	if !started {
		// 异步体未启动（启动前的回调/单测未装配）：不能默默丢弃，明确报错给调用方
		return errorx.NewBizI(ctx, imsg.ErrProposalExecutorNotStarted)
	}
	select {
	case queue <- proposalId:
		return nil
	case <-stop:
		return errorx.NewBizI(ctx, imsg.ErrProposalExecutorStopped)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// execWorker 受控执行体。recover 收在「单条提案」粒度：收在函数出口会让一次 panic 永久少一个
// 执行者（同 RunService worker 的教训），而已领取的提案会停在 approved 无人收尾
func (g *proposalGatewayImpl) execWorker() {
	defer g.execWg.Done()
	for {
		select {
		case <-g.execStop:
			return
		case proposalId := <-g.execQueue:
			g.processApproved(proposalId)
		}
	}
}

// processApproved 处理一条已领取的提案：任何 panic 都必须落终态，否则提案永久停在 approved（无驱动路径）
func (g *proposalGatewayImpl) processApproved(proposalId uint64) {
	var panicErr error
	func() {
		defer gox.Recover(func(e error) { panicErr = e })
		proposal, err := g.GetById(proposalId)
		if err != nil || proposal == nil {
			logx.Errorf("[agentops-proposal] queued proposal not found before execution: proposalId=%d", proposalId)
			return
		}
		g.executeApproved(contextx.WithTraceId(context.Background()), proposal)
	}()
	if panicErr == nil {
		return
	}
	ctx := contextx.WithTraceId(context.Background())
	logx.ErrorfContext(ctx, "[agentops-proposal] execution panicked (recovered, proposal marked failed): proposalId=%d, err=%v", proposalId, panicErr)
	if rows, err := g.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().
			Set("status", entity.ProposalStatusFailed).
			Set("err", truncateProposalErr(fmt.Sprintf("execution panicked: %v", panicErr))),
		model.NewCond().EqStrict("id", proposalId).EqStrict("status", entity.ProposalStatusApproved)); err != nil {
		logx.ErrorfContext(ctx, "[agentops-proposal] mark panicked proposal failed: proposalId=%d, err=%v", proposalId, err)
	} else if rows == 1 {
		// panic 也是执行终态：抢到终态写入才回写工单（落败说明别的路径已收尾，结果由那条路径回写）
		g.recordTicketResult(ctx, proposalId)
	}
}

// executeApproved 受控执行体的工作：解析一次性许可 → 执行档按锁定参数执行 → 只读校验档复查 → 回写终态。
//
// 在异步 worker 的 goroutine 里同步跑（不再占据审批请求链路），但必须以终态收尾（executed/failed）：
// 停在 approved 就只能等停滞清扫重投（见 SweepStalledProposals），而重投只能靠幂等键短路，
// 既白跑一轮也让治理看板的悬挂计数一直不归零
func (g *proposalGatewayImpl) executeApproved(ctx context.Context, proposal *entity.Proposal) {
	ownerAccountId, ownerUsername, notifyType, notifyTmplCode, timeoutSeconds := g.resolveExecConfig(ctx, proposal)
	pidStr := strconv.FormatUint(proposal.Id, 10)

	// 0) 把锁定的 action 解析成本次运行只允许调用的工具（机制级约束，不只是提示词）
	allowed, err := allowedToolsForAction(ctx, proposal.Action, proposal.Params)
	if err != nil {
		g.markFailed(ctx, proposal, nil, err)
		logx.ErrorfContext(ctx, "[agentops-proposal] cannot resolve run grant, execution refused: proposalId=%d, action=%q, err=%v",
			proposal.Id, proposal.Action, err)
		return
	}

	// 1) 受控执行：executor 档，参数锁定为提案 params，工具限定为许可内的那一个
	execRun, execErr := g.runService.RunSync(ctx, &EnqueueReq{
		TriggerType:      entity.TriggerTypeManual,
		TriggerRef:       "proposal:" + pidStr,
		ProfileId:        headless.ProfileExecutor,
		Title:            proposal.Title,
		RunRole:          entity.RunRoleExec,
		Instruction:      proposalExecInstruction,
		Input:            buildExecutorInput(proposal),
		OwnerAccountId:   ownerAccountId,
		OwnerUsername:    ownerUsername,
		TimeoutSeconds:   timeoutSeconds,
		NotifyType:       notifyType,
		NotifyTmplCode:   notifyTmplCode,
		IdempotencyKey:   "proposal-exec:" + pidStr,
		AllowedToolNames: allowed,
		// 把「人已批准的参数」一并下发：受控执行那轮里工具默认还要二次审批，而无头运行没有
		// 应答者——不绑定就根本执行不了；绑定参数才能既放行「批准的那一组」、又拦住模型换参
		//（批准的是 command:"true"，跑成别的就不给执行）
		ApprovedActions: map[string]string{proposal.Action: proposal.Params},
	})

	// 幂等键短路撞上在途运行：RunSync 命中幂等键时原样返回既有运行，其状态可能仍是 running
	//（多实例部署下另一实例正在执行，或本次是停滞重投而上一任执行者尚未退出）。
	// 这不是失败：若按「状态 != succeeded」归入 execFailed，会把一条正在执行的提案标成 failed，
	// 而真实执行稍后成功落地——「提案说失败、变更其实生效」比停在 approved 更误导（approved
	// 至少还有停滞清扫会回来看，failed 是终态、无人再管，而且工单会被连带改判成处理失败）
	if execRun != nil && !entity.IsTerminal(execRun.Status) {
		logx.InfofContext(ctx, "[agentops-proposal] exec run still in flight, leave the proposal to its owner: proposalId=%d, execRunId=%d, status=%s",
			proposal.Id, execRun.Id, execRun.Status)
		return
	}

	// 失败判定不能只看 run 状态：模型可以把工具报错诚实写进回复，而那轮 LLM 调用本身成功
	//（实测：受控执行因库名错误报 Access denied，run=succeeded、提案却被记成 executed）。
	// 「变更到底有没有发生」的唯一事实来源是工具级结果
	execFailed := execErr != nil || execRun == nil || execRun.Status != entity.RunStatusSucceeded || len(execRun.ToolErrors()) > 0
	if execFailed {
		g.markFailed(ctx, proposal, execRun, execErr)
		errMsg := "execution did not succeed"
		if execErr != nil {
			errMsg = execErr.Error()
		} else if execRun != nil {
			if toolErrs := execRun.ToolErrors(); len(toolErrs) > 0 {
				errMsg = "tool execution failed: " + toolErrs[0]
			} else {
				errMsg = "exec run status: " + execRun.Status
			}
		}
		logx.WarnfContext(ctx, "[agentops-proposal] execution failed: proposalId=%d, err=%s", proposal.Id, errMsg)
		return
	}

	// 执行结果回写（先落执行结果，再校验，保证执行事实不丢）
	execResult := ""
	if execRun != nil {
		execResult = execRun.Summary
	}
	g.patchProposal(ctx, proposal.Id, base.NewPatch().
		Set("execRunId", execRun.Id).
		Set("execResult", execResult))

	// 2) 只读校验：reviewer 档复查执行效果（best-effort，校验失败不影响「已执行」事实）
	verifyRun, verifyErr := g.runService.RunSync(ctx, &EnqueueReq{
		TriggerType:    entity.TriggerTypeManual,
		TriggerRef:     "proposal-verify:" + pidStr,
		ProfileId:      headless.ProfileReviewer,
		Title:          proposal.Title,
		RunRole:        entity.RunRoleVerify,
		Instruction:    proposalVerifyInstruction,
		Input:          buildVerifyInput(proposal, execResult),
		OwnerAccountId: ownerAccountId,
		OwnerUsername:  ownerUsername,
		TimeoutSeconds: timeoutSeconds,
		IdempotencyKey: "proposal-verify:" + pidStr,
	})
	verifyConclusion := ""
	if verifyRun != nil {
		verifyConclusion = verifyRun.Summary
	}
	if verifyErr != nil {
		logx.WarnfContext(ctx, "[agentops-proposal] verify run failed (non-fatal): proposalId=%d, err=%v", proposal.Id, verifyErr)
	}

	// 3) 回写终态：executed + 校验结论（真实结果同步回写审批工单，回执只停留在校验通过那一程）
	patch := base.NewPatch().
		Set("status", entity.ProposalStatusExecuted).
		Set("execResult", execResult).
		Set("verifyConclusion", verifyConclusion)
	if verifyRun != nil {
		patch = patch.Set("verifyRunId", verifyRun.Id)
	}
	g.patchProposal(ctx, proposal.Id, patch)
	g.recordTicketResult(ctx, proposal.Id)

	logx.InfofContext(ctx, "[agentops-proposal] executed & verified: proposalId=%d, execRunId=%d, verifyRunId=%d",
		proposal.Id, execRun.Id, idOf(verifyRun))
}

// validateProposalAction 提案 action 的准入判据（登记侧 Record 与许可下发侧 allowedToolsForAction 共用）。
//
// 必须只有一处定义：两处强度不一致时，会出现「存得进、批得了、跑不了」——登记时不拦而等
// 批准后才拒，审批人已白做一轮判断。入参 name 由调用方保证非空（两侧的「空动作」语义不同：
// 登记侧是「缺必填字段」，下发侧是「无法确定许可范围」，错误文案因此留在各自入口）。
//
// 判据取能力表而不是工具名名单：新增工具只需在注册处声明能力，本函数零改动（开闭原则）。
func validateProposalAction(ctx context.Context, name, params string) error {
	if !aitools.IsDeclaredTool(name) {
		return errorx.NewBizI(ctx, imsg.ErrProposalActionNotTool, "action", name)
	}
	// action 必须是「会产生变更」的工具。只读工具（含 propose_remediation 本身，它在能力表里
	// 就是 CapabilityRead）作为 action 时，受控执行那轮不会改动任何状态，提案却被记成 executed，
	// 留下一条「已执行但什么都没发生」的审计事实（比记成失败更误导）；而允许 propose_remediation
	// 等于「一条已批准的提案可以直接登记出另一条提案」，其 action/params 完全由提案内容决定，
	// 审批链被提案造提案稀释
	if aitools.CapabilityOf(name) == aitools.CapabilityRead {
		return errorx.NewBizI(ctx, imsg.ErrProposalActionNotMutable, "action", name)
	}
	// 参数形状校验（属性集由工具侧反射声明，见 tools.DeclareParamSchema）。
	//
	// 不校验会怎样：提案能登记、能被批准，但受控执行那轮的参数绑定永远对不上
	// （实测到提案里写的是 cmd 而工具收 command），最终表现为「批准后什么都没发生」
	// 却无任何报错。在这里冒错，坏数据才会在入口就被拦住
	if err := aitools.ValidateToolParams(name, params); err != nil {
		return errorx.NewBiz(err.Error())
	}
	return nil
}

// allowedToolsForAction 把提案锁定的 action 解析为「本次运行只允许调用这些工具」的一次性许可。
//
// 解析不出来（action 是语义动作而不是工具名）时拒绝执行而不是放开全量工具：受控执行档声明了
// 「必须由许可驱动」，若为了兼容语义动作而省略许可，「审批锁定那一个动作」就只剩提示词约束，
// 护栏反而成了唯一拦住模型的东西（而它已无约束可查）。工具名集合的真源是能力表（见 tools.IsDeclaredTool）。
func allowedToolsForAction(ctx context.Context, action, params string) ([]string, error) {
	name := strings.TrimSpace(action)
	if name == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrProposalNoActionForGrant)
	}
	if err := validateProposalAction(ctx, name, params); err != nil {
		return nil, err
	}
	return []string{name}, nil
}

// resolveExecConfig 解析执行/校验运行的身份与通知配置：优先用提案归属，其次回退到来源诊断运行
// 的通知配置（复用同一消息模板，使执行结果沿用原任务/触发器的通知渠道）。
func (g *proposalGatewayImpl) resolveExecConfig(ctx context.Context, proposal *entity.Proposal) (ownerAccountId uint64, ownerUsername string, notifyType int8, notifyTmplCode string, timeoutSeconds int) {
	ownerAccountId = proposal.OwnerAccountId
	ownerUsername = proposal.OwnerUsername
	if proposal.RunId == 0 {
		return
	}
	srcRun, err := g.runService.GetById(proposal.RunId)
	if err != nil || srcRun == nil {
		return
	}
	notifyType = srcRun.NotifyType
	notifyTmplCode = srcRun.NotifyTmplCode
	timeoutSeconds = srcRun.TimeoutSeconds
	if ownerAccountId == 0 {
		ownerAccountId = srcRun.OwnerAccountId
		ownerUsername = srcRun.OwnerUsername
	}
	return
}

// markFailed 回写提案执行失败终态（含执行运行关联与错误原因）
func (g *proposalGatewayImpl) markFailed(ctx context.Context, proposal *entity.Proposal, execRun *entity.Run, execErr error) {
	errMsg := ""
	switch {
	case execErr != nil:
		errMsg = execErr.Error()
	case execRun != nil && len(execRun.ToolErrors()) > 0:
		// 工具级失败优先：它是「没干成」的直接证据，而 run.Status 可能是 succeeded
		errMsg = strings.Join(execRun.ToolErrors(), "; ")
	case execRun != nil:
		errMsg = "exec run status: " + execRun.Status
		if execRun.Err != "" {
			errMsg = execRun.Err
		}
	}
	patch := base.NewPatch().
		Set("status", entity.ProposalStatusFailed).
		Set("err", truncateProposalErr(errMsg))
	if execRun != nil {
		patch = patch.Set("execRunId", execRun.Id).Set("execResult", execRun.Summary)
	}
	g.patchProposal(ctx, proposal.Id, patch)
	// 执行失败同样要回写工单：否则工单永远显示「已受理」，失败原因只存在提案页里
	g.recordTicketResult(ctx, proposal.Id)
}

// patchProposal best-effort 补丁更新提案列（失败仅记日志，不阻断主流程）
func (g *proposalGatewayImpl) patchProposal(ctx context.Context, proposalId uint64, patch base.Patch) {
	if _, err := g.GetRepo().UpdateColsByCond(ctx, patch, model.NewCond().EqStrict("id", proposalId)); err != nil {
		logx.ErrorfContext(ctx, "[agentops-proposal] patch proposal failed: id=%d, err=%v", proposalId, err)
	}
}

// SyncProcStatus 惰性对账：按审批工单的终态把仍停在「待审批」的提案校准到应有状态（CAS，避免覆盖已流转状态）。
//
// flowEventBus 为 flow 内部总线（agentops 不可订阅），且驳回/取消不触发 FlowBizHandle 回调，
// 故在读取路径 best-effort 对账，使提案状态自愈（不残留永久 pending）。「退回/挂起」不视为
// 终态驳回（可能被重新提交），保持 pending。
func (g *proposalGatewayImpl) SyncProcStatus(ctx context.Context, proposals []*entity.Proposal) {
	if len(proposals) == 0 {
		return
	}
	procinstApp := flowProcinstFn()
	if procinstApp == nil {
		return
	}
	for _, p := range proposals {
		if p == nil || p.Status != entity.ProposalStatusPending || p.ProcInstId == 0 {
			continue
		}
		procinst, err := procinstApp.GetById(p.ProcInstId)
		if err != nil || procinst == nil {
			continue
		}
		target := reconciledProposalStatus(procinst.Status)
		if target == "" {
			continue
		}
		if _, uerr := g.GetRepo().UpdateColsByCond(ctx,
			base.NewPatch().Set("status", target),
			model.NewCond().EqStrict("id", p.Id).EqStrict("status", entity.ProposalStatusPending)); uerr != nil {
			logx.WarnfContext(ctx, "[agentops-proposal] sync proposal status failed: id=%d, procInstStatus=%d, err=%v", p.Id, procinst.Status, uerr)
			continue
		}
		p.Status = target
	}
}

// reconciledProposalStatus 工单状态 → 仍停在 pending 的提案应对账到的状态（空串=不对账）。
//
// 三个方向的取值都由「工单上已经是事实的审批结果」推出，而不是由回调是否跑过推出：
//   - 取消/终止：审批未通过 → rejected
//   - 完成：审批已通过 → approved。这一支必须存在，否则「工单说已批准且处理成功、提案说待审批」
//     两个权威视图长期互相矛盾（实测到一条提案停在 pending 五天，而它的工单早已完成并写着
//     execution=queued）：治理看板的待审批数只增不减，提案又因 pending 非终态而删不掉，
//     而 flow 侧的恢复入口（清扫/改期/取消执行都只认业务待处理）对这类工单全部不可达。
//     对账成 approved 后它落入既有悬挂不变量（approved 且无执行运行），由停滞清扫接手重投。
//     注意这里只改状态、不触发执行：本函数在读路径上，不得产生「对目标机器下发命令」这类副作用
//   - 退回/挂起/进行中：审批可被重新提交，提案仍活着 → 保持 pending
//
// 与审批回调并发时也是安全的：两边都用「仍为 pending」做 CAS，判赢的一方去执行，判败的一方走
// handleClaimLost（它恰好会把「approved 且无执行运行」重新投递），故不会漏执行也不会双执行
func reconciledProposalStatus(procinstStatus flowentity.ProcinstStatus) string {
	switch procinstStatus {
	case flowentity.ProcinstStatusCancelled, flowentity.ProcinstStatusTerminated:
		return entity.ProposalStatusRejected
	case flowentity.ProcinstStatusCompleted:
		return entity.ProposalStatusApproved
	default:
		return ""
	}
}

// stalledProposalCond 「已领取执行权但受控执行从未落库」的停滞判据（停滞清扫与治理看板共用）。
//
// 必须共用一份：两边各写一遍就会漂移，于是看板报红的条数与清扫真正会去动的条数不是同一批，
// 运维盯着的那个数字不再等于系统的实际行为——那比没有这个指标更坏。
//
// 静默期条件不可省：exec_run_id 要等执行档返回终态才回写，故正常在途的执行同样是
// 「approved 且 exec_run_id=0」。少了它，每一次正常执行期间看板都会把这条提案计成悬挂并打红，
// 而假警报的代价是真出问题时已经没人再看这个指标
func stalledProposalCond() *model.QueryCond {
	return model.NewCond().
		EqStrict("status", entity.ProposalStatusApproved).
		EqStrict("execRunId", 0).
		And("update_time < ?", time.Now().Add(-proposalStallAfter))
}

// SweepStalledProposals 重投「已领取执行权但受控执行从未落库」的提案（approved 且 exec_run_id=0）。
//
// 为什么必须有这条自愈路径：受控执行体投递的是**进程内队列**，进程在 worker 取出前退出即丢单；
// 而工单在回调返回「已受理」那一刻就被 flow 记为处理成功，于是 flow 侧所有恢复入口（清扫只捞
// 业务待处理、改期/取消执行同样要求待处理、卡住告警只看执行中）对这类工单全部永久不可达，
// FlowBizHandle 也不会再被调用——handleClaimLost 的兜底因此永无触发时机。不重投的后果：
// 提案永久停在 approved，治理看板的悬挂计数只增不减，而变更其实早已批准。
//
// 重投是安全的（与 flow 拒绝自动重放「执行中」行的取向并不冲突，那里没有幂等键可用）：
//   - 从未执行过 → 正常执行，落地既已批准的变更
//   - 执行过但进程死在回写关联之前 → 幂等键命中，RunSync 短路返回既有运行，不重复执行
//   - 执行仍在途（多实例部署） → executeApproved 按非终态识别并放弃本次，留待其归属者收尾
func (g *proposalGatewayImpl) SweepStalledProposals(ctx context.Context) (int, error) {
	stalled, err := g.GetRepo().SelectByCond(stalledProposalCond(), "id")
	if err != nil {
		return 0, err
	}
	if len(stalled) > proposalStallSweepBatch {
		logx.WarnfContext(ctx, "[agentops-proposal] %d stalled proposals exceed sweep batch %d, the rest will be retried on the next sweep",
			len(stalled), proposalStallSweepBatch)
		stalled = stalled[:proposalStallSweepBatch]
	}
	requeued := 0
	for _, p := range stalled {
		if !g.trySubmitForExecution(p.Id) {
			// 队列满/执行体未启动/已停机：本轮停止，剩下留到下一轮清扫。
			// 不能等：等下去不会报错，只会把调用方（含 Init）拖住（见 trySubmitForExecution）
			logx.WarnfContext(ctx, "[agentops-proposal] requeue stalled proposal not accepted (queue full or executor stopped), stop sweeping this round: proposalId=%d, remaining=%d",
				p.Id, len(stalled)-requeued)
			break
		}
		requeued++
		logx.WarnfContext(ctx, "[agentops-proposal] requeued stalled approved proposal (execution never landed): proposalId=%d", p.Id)
	}
	return requeued, nil
}

// stallSweeper 周期重投停滞提案（语义见 SweepStalledProposals）。
//
// 独立于 RunService 的租约清扫：那是运行级的收敛（把无人管的 running 置败），这里是提案级的
// 「已批准但执行从未落库」（重新推进），判据与恢复动作都不同，合并会把两种相反的方向弄反
func (g *proposalGatewayImpl) stallSweeper() {
	defer g.execWg.Done()

	ticker := time.NewTicker(proposalStallSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-g.execStop:
			return
		case <-ticker.C:
			ctx := contextx.WithTraceId(context.Background())
			if _, err := g.SweepStalledProposals(ctx); err != nil {
				logx.ErrorfContext(ctx, "[agentops-proposal] stalled proposal sweep failed: %v", err)
			}
		}
	}
}

// 执行档/校验档的任务级 system 指令（与 Profile 静态指令叠加，强调「参数锁定」与「只读复查」）
const (
	proposalExecInstruction = `本次运行是「变更提案经人工审批通过后的受控执行」。
硬性约束：
1. 只能执行下方「已审批变更提案」中锁定的动作与参数，不得扩大范围、不得改动参数、不得执行任何额外变更；
2. 调用相应工具（如 ExecSql / MachineCommandExec 等）时，严格使用锁定参数；
3. 执行完成后如实回报：是否成功、关键输出、影响的资源；若失败，说明失败原因。`

	proposalVerifyInstruction = `本次运行是「变更执行后的只读校验」。
职责：仅使用只读工具复查目标资源状态/指标，判定预期目标是否达成、是否出现副作用；
未达成或出现副作用时给出明确的回滚建议。不得执行任何变更。`
)

// buildExecutorInput 构造执行档运行输入（user 消息）：把锁定参数原样下发，禁止模型改写
func buildExecutorInput(proposal *entity.Proposal) string {
	var sb strings.Builder
	sb.WriteString("已审批通过的变更提案：\n")
	sb.WriteString("标题：" + proposal.Title + "\n")
	sb.WriteString("动作：" + proposal.Action + "\n")
	sb.WriteString("锁定参数(JSON)：" + proposal.Params + "\n")
	sb.WriteString("理由：" + proposal.Reason + "\n\n")
	sb.WriteString("请严格按上述锁定参数执行该动作，并回报执行结果（成功/失败/关键输出）。")
	return sb.String()
}

// buildVerifyInput 构造校验档运行输入（user 消息）：携带执行结果供只读复查
func buildVerifyInput(proposal *entity.Proposal, execResult string) string {
	var sb strings.Builder
	sb.WriteString("以下变更提案已审批通过并已执行，请只读复查执行效果：\n")
	sb.WriteString("标题：" + proposal.Title + "\n")
	sb.WriteString("动作：" + proposal.Action + "\n")
	sb.WriteString("锁定参数(JSON)：" + proposal.Params + "\n")
	sb.WriteString("预期理由：" + proposal.Reason + "\n")
	sb.WriteString("执行结果摘要：" + execResult + "\n\n")
	sb.WriteString("请复查目标资源状态/指标，判定预期目标是否达成、是否出现副作用；未达成或有副作用时给出回滚建议。")
	return sb.String()
}

// idOf 取运行 id（nil 安全）
func idOf(run *entity.Run) uint64 {
	if run == nil {
		return 0
	}
	return run.Id
}

// maxProposalErrLen err 列截断长度（与实体 size:2000 对齐）
const maxProposalErrLen = 2000

// truncateProposalErr 截断错误信息至列宽，避免写库溢出
func truncateProposalErr(err string) string {
	rs := []rune(err)
	if len(rs) <= maxProposalErrLen {
		return err
	}
	return string(rs[:maxProposalErrLen])
}
