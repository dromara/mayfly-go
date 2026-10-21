package tools

import (
	"context"
	"errors"
	"strings"
	"sync"

	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// propose_remediation 变更提案工具（自主运维闭环的「提案」环节，capability=read）。
//
// 只读诊断档也可用：Agent 不直接执行变更，而是把「要做什么（action）+ 锁定参数（params）
// + 理由（reason）+ 风险级（risk）」登记为一条结构化变更提案，交由人工审批（flow 工单）
// 通过后由执行档受控执行。这样只读档在没有任何写权限的前提下，仍能推动变更闭环，
// 且每一步变更都有人工审批关卡（对齐安全基线：默认只读诊断，变更须审批）。
//
// 依赖倒置（破除 ai → agentops 反向依赖）：提案的持久化与审批发起属于 agentops 职责，
// 本工具仅面向 ProposalRecorder 端口编程；agentops 在 init 期经 SetProposalRecorder 注入实现。
// 未注入时（如未装配 agentops）工具优雅降级，回传「提案通道未就绪」提示而非 panic。

// 提案风险级（与 agentops 提案实体口径一致；此处以字符串契约约束模型取值）
const (
	ProposalRiskLow      = "low"
	ProposalRiskMedium   = "medium"
	ProposalRiskHigh     = "high"
	ProposalRiskCritical = "critical"
)

// ProposalDraft 变更提案草稿（工具采集 + 运行上下文关联，交给 Recorder 落库/发起审批）
type ProposalDraft struct {
	// RunId 发起提案的无头运行 id（agentops Run.Id；无关联时为 0）
	RunId uint64
	// ConversationId 关联会话 id（回放入口；无痕运行时为 0）
	ConversationId uint64
	// Title 提案标题（一句话概述变更）
	Title string
	// Action 变更动作：必须是可调用的工具名（登记时校验，审批后只能执行这一个工具）
	Action string
	// Params 锁定参数 JSON（审批通过后执行档严格按此参数执行，不得自由发挥）
	Params string
	// Risk 风险级（low/medium/high/critical）
	Risk string
	// Reason 提案理由（为什么需要这个变更、预期效果）
	Reason string
	// OwnerAccountId 运行服务身份账号 id（提案归属/审批发起人）
	OwnerAccountId uint64
	// OwnerUsername 运行服务身份账号名
	OwnerUsername string
}

// ProposalRecord 提案登记结果（回传给模型，让其知晓提案已受理及审批状态）
type ProposalRecord struct {
	// ProposalId 提案 id
	ProposalId uint64
	// ProcInstId 审批流程实例 id（未绑定审批流时为 0）
	ProcInstId uint64
	// Status 提案状态（pending/approved/rejected/executed/failed）
	Status string
	// Message 面向模型的补充说明（如「已发起审批工单，等待人工审批」/「未绑定审批流」）
	Message string
}

// ProposalRecorder 变更提案登记端口（由 agentops 实现并在 init 期注入）
type ProposalRecorder interface {
	// Record 登记一条变更提案（落库 + 视配置发起 flow 审批工单），返回登记结果
	Record(ctx context.Context, draft *ProposalDraft) (*ProposalRecord, error)
}

var (
	proposalRecorderMu sync.RWMutex
	proposalRecorder   ProposalRecorder
)

// SetProposalRecorder 注入变更提案登记实现（agentops init 期调用；后注册覆盖先注册）。
//
// 时序契约：须在首次工具调用前注入（agentops 经 starter.AddInitFunc 在启动期完成）；
// 未注入时工具优雅降级，不影响 Agent 其余能力。
func SetProposalRecorder(r ProposalRecorder) {
	proposalRecorderMu.Lock()
	defer proposalRecorderMu.Unlock()
	proposalRecorder = r
}

// getProposalRecorder 读取当前注入的登记实现（可能为 nil）
func getProposalRecorder() ProposalRecorder {
	proposalRecorderMu.RLock()
	defer proposalRecorderMu.RUnlock()
	return proposalRecorder
}

// ProposeRemediationParam propose_remediation 工具入参
type ProposeRemediationParam struct {
	// Title 提案标题（一句话概述要做的变更）
	Title string `json:"title" jsonschema_description:"提案标题，一句话概述要执行的变更，如「重启 web-1 的 nginx 服务」" jsonschema:"required"`
	// Action 变更动作（必须是可调用的工具名，登记时校验）
	Action string `json:"action" jsonschema_description:"变更动作：必须是本系统可调用的**会产生变更的**工具名（如 ExecSql、MachineCommandExec）。审批通过后受控执行只允许调用这一个工具，因此两类值会被拒绝登记：① 非工具名的语义动作（如 restart-service）；② 只读查询工具（如 DbQueryData、ListResources、QueryMetrics）与 propose_remediation 本身——它们不改动任何状态，登记后只会得出一条「已执行但什么都没发生」的假记录。若只是建议人工处理或只读探查，请写进诊断结论而不是登记为提案" jsonschema:"required"`
	// Params 锁定参数 JSON（审批通过后执行档严格按此参数执行）
	Params string `json:"params" jsonschema_description:"锁定参数的 JSON 字符串：键名必须与 action 工具自身的入参名完全一致，不得自造键名、也不得省略必填项（登记时服务端会按该工具的参数 schema 校验，不合法会在报错里列出合法属性名）。审批通过后执行档将严格按此参数执行，不得改动" jsonschema:"required"`
	// Risk 风险级
	Risk string `json:"risk" jsonschema_description:"风险级，取值之一：low（低危只读性变更）/medium（中危）/high（高危，影响面大）/critical（极高危，不可逆）" jsonschema:"required,enum=low;medium;high;critical"`
	// Reason 提案理由
	Reason string `json:"reason" jsonschema_description:"提案理由：为什么需要这个变更、基于哪些诊断证据、预期达到什么效果（供审批人判断）" jsonschema:"required"`
}

// ProposeRemediationOutput propose_remediation 工具出参
type ProposeRemediationOutput struct {
	// ProposalId 提案 id
	ProposalId uint64 `json:"proposalId"`
	// ProcInstId 审批流程实例 id（0 表示未绑定审批流）
	ProcInstId uint64 `json:"procInstId"`
	// Status 提案状态
	Status string `json:"status"`
	// Message 补充说明
	Message string `json:"message"`
}

// GetProposeRemediation 创建 propose_remediation 工具
//
// 经 agent/ext/proposaltool 的 ToolContributor 通道贡献给 Agent（不全局注册）。
func GetProposeRemediation() (tool.InvokableTool, error) {
	return utils.InferTool("propose_remediation",
		"登记一条变更提案（用于需要写操作/高危操作的场景）。当前为只读诊断模式，你不能直接执行任何变更；"+
			"若诊断结论需要变更（执行 SQL、下发机器命令、重启服务、改配置等），必须调用本工具把「操作+锁定参数+理由+风险级」"+
			"登记为变更提案，交由人工审批通过后由受控执行档执行。本工具不会立即执行任何变更，仅登记提案。",
		func(ctx context.Context, param *ProposeRemediationParam) (*ProposeRemediationOutput, error) {
			if strings.TrimSpace(param.Action) == "" {
				return nil, NewToolError(errors.New("action is required (must be a callable tool name, e.g. ExecSql / MachineCommandExec)"), RecoverRetry)
			}
			if strings.TrimSpace(param.Params) == "" {
				return nil, NewToolError(errors.New("params is required (locked execution parameters as JSON)"), RecoverRetry)
			}
			if strings.TrimSpace(param.Reason) == "" {
				return nil, NewToolError(errors.New("reason is required (justify the change for the approver)"), RecoverRetry)
			}

			recorder := getProposalRecorder()
			if recorder == nil {
				// 提案通道未就绪（agentops 未装配）：优雅降级，明确告知模型而非 panic/静默失败
				logx.WarnfContext(ctx, "[propose_remediation] proposal recorder not registered; cannot record proposal")
				return &ProposeRemediationOutput{
					Status:  "unavailable",
					Message: "变更提案通道当前未就绪（运维模块未装配），提案未登记。请仅产出只读诊断结论与建议措施，稍后由运维人员手动处理。",
				}, nil
			}

			// 运行身份（提案归属/审批发起人）+ 业务关联（回链所属 Run/会话）
			draft := &ProposalDraft{
				Title:  strings.TrimSpace(param.Title),
				Action: strings.TrimSpace(param.Action),
				Params: NormalizeActionParams(ctx, param.Action, param.Params),
				Risk:   normalizeRisk(param.Risk),
				Reason: strings.TrimSpace(param.Reason),
			}
			if la := contextx.GetLoginAccount(ctx); la != nil {
				draft.OwnerAccountId = la.Id
				draft.OwnerUsername = la.Username
			}
			if assoc := headless.RunAssocFrom(ctx); assoc != nil {
				draft.RunId = assoc.BizRunId
				draft.ConversationId = assoc.ConversationId
			}

			record, err := recorder.Record(ctx, draft)
			if err != nil {
				// 登记失败属可重试类（如瞬时 DB/流程错误）：退回模型让其知晓并重试或改走只读结论
				return nil, NewToolError(err, RecoverRetry)
			}
			out := &ProposeRemediationOutput{Status: record.Status, Message: record.Message}
			out.ProposalId = record.ProposalId
			out.ProcInstId = record.ProcInstId
			return out, nil
		},
	)
}

// ProposeRemediationTools 变更提案工具集（供 ToolContributor 聚合）
func ProposeRemediationTools() ([]tool.BaseTool, error) {
	t, err := GetProposeRemediation()
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{t}, nil
}

// normalizeRisk 归一风险级：非法/空值按 high 保守处理（宁可高估风险走更严审批）
func normalizeRisk(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case ProposalRiskLow:
		return ProposalRiskLow
	case ProposalRiskMedium:
		return ProposalRiskMedium
	case ProposalRiskCritical:
		return ProposalRiskCritical
	case ProposalRiskHigh:
		return ProposalRiskHigh
	default:
		return ProposalRiskHigh
	}
}
