package entity

import (
	"mayfly-go/pkg/model"

	"gorm.io/gorm"
)

// Proposal 变更提案：自主运维闭环的「提案」环节聚合。
//
// 只读诊断档经 propose_remediation 工具登记一条结构化提案（要做什么 action + 锁定参数 params
// + 理由 reason + 风险级 risk），由 ProposalGateway 发起 flow 审批工单（proc_inst_id）。审批通过
// 回调 → 执行档按锁定参数受控执行（exec_run_id）→ 只读校验档复查（verify_run_id）→ 结论回写。
//
// 提案是「诊断 → 提案 → 审批 → 受控执行 → 验证」闭环的事实源：状态机、审批工单关联、
// 执行/校验运行关联、执行结果与校验结论都在此聚合，替代散落的多套记录。
//
// 用 model.Model（含逻辑删除）：提案是用户可见可清理的记录实体，删除留痕与任务/触发器口径一致。
type Proposal struct {
	model.Model
	// ExtraData 展示型辅助字段的容器（owner 账号名等不进查询/排序的字段收敛在这里）
	model.ExtraData

	// 以下带 form tag 的字段就是列表页的查询维度，与 proposalRepoImpl.GetPageList 的过滤条件一一对应。
	// form tag 不可省：Gin 的 BindQuery 只认 form，缺失时退回**字段名**（大小写敏感），
	// 于是 ?status=executed 绑不进 Status——请求照发、接口照返 200、过滤条件被静默丢弃、列表返回全量，
	// 而界面上的搜索表单还在，用户只会以为「没有匹配的数据」
	// RunId 发起提案的无头运行 id（agentops Run.Id；无关联时为 0）
	RunId uint64 `json:"runId" form:"runId" gorm:"index:idx_aiops_proposal_run;comment:来源运行id"`
	// ConversationId 关联会话 id（回放入口；无痕运行时为 0）
	ConversationId uint64 `json:"conversationId" gorm:"index:idx_aiops_proposal_conv;comment:关联会话id"`

	// Title 提案标题（一句话概述变更）
	Title string `json:"title" form:"title" gorm:"size:255;comment:提案标题"`
	// Action 变更动作：必须是可调用的工具名（登记时校验）。审批后受控执行把它解析成本次
	// 运行的一次性工具许可，只允许调用这一个工具
	Action string `json:"action" form:"action" gorm:"size:128;not null;comment:变更动作（工具名）"`
	// Params 锁定参数 JSON（审批通过后执行档严格按此参数执行，不得自由发挥）
	Params string `json:"params" gorm:"type:text;comment:锁定参数JSON"`
	// Risk 风险级（low/medium/high/critical；见 ProposalRisk* 常量）
	Risk string `json:"risk" form:"risk" gorm:"size:16;not null;index:idx_aiops_proposal_risk;comment:风险级"`
	// Reason 提案理由（为什么需要这个变更、诊断依据、预期效果）
	Reason string `json:"reason" gorm:"type:text;comment:提案理由"`

	// Status 提案状态（pending/approved/rejected/executed/failed；见 ProposalStatus* 常量）
	Status string `json:"status" form:"status" gorm:"size:24;not null;index:idx_aiops_proposal_status;comment:提案状态"`

	// OwnerAccountId 运行服务身份账号 id（提案归属/审批发起人；工具权限按此收敛）
	OwnerAccountId uint64 `json:"ownerAccountId" form:"ownerAccountId" gorm:"not null;index:idx_aiops_proposal_owner;comment:归属账号id"`
	// OwnerUsername 运行服务身份账号名（展示/审计）。瞬态字段：落库收敛在 Extra（非查询维度）
	OwnerUsername string `json:"ownerUsername" gorm:"-"`

	// ProcInstId 审批流程实例 id（0=未绑定审批流；审批状态对账/回调据此定位提案）
	ProcInstId uint64 `json:"procInstId" form:"procInstId" gorm:"index:idx_aiops_proposal_proc;comment:审批流程实例id"`
	// ProcDefId 审批流程定义 id（发起工单时快照）
	ProcDefId uint64 `json:"procDefId" gorm:"comment:审批流程定义id"`

	// ExecRunId 受控执行运行 id（审批通过后执行档产出；0=尚未执行）
	ExecRunId uint64 `json:"execRunId" gorm:"comment:执行运行id"`
	// VerifyRunId 只读校验运行 id（执行后复查产出；0=尚未校验）
	VerifyRunId uint64 `json:"verifyRunId" gorm:"comment:校验运行id"`

	// ExecResult 执行结果摘要（执行档最终输出）
	ExecResult string `json:"execResult" gorm:"type:text;comment:执行结果摘要"`
	// VerifyConclusion 校验结论（校验档复查结论：目标是否达成/是否需回滚）
	VerifyConclusion string `json:"verifyConclusion" gorm:"type:text;comment:校验结论"`

	// Err 执行/校验失败原因（终态非成功时填充）
	Err string `json:"err" gorm:"size:2000;comment:错误信息"`
}

// TableName 指定表名
func (p *Proposal) TableName() string {
	return "t_aiops_proposal"
}

// BeforeSave 把展示型瞬态字段收进 Extra（Extra 是唯一落库真源），钩子形参说明见 AgentTask.BeforeSave
func (p *Proposal) BeforeSave(*gorm.DB) error {
	p.SetExtraValue(ExtraKeyOwnerUsername, p.OwnerUsername)
	return nil
}

// AfterFind 从 Extra 还原瞬态展示字段（VO 按名拷贝与 JSON 输出保持原契约）
func (p *Proposal) AfterFind(*gorm.DB) error {
	p.OwnerUsername = p.GetExtraString(ExtraKeyOwnerUsername)
	return nil
}

// 提案状态机
//
//	pending ──(审批通过回调，CAS 领取执行权)──> approved ──(异步受控执行)──> executed
//	   │                          │                                       └──> failed
//	   │                          └─(停滞清扫重投: approved 且无执行运行)─┘
//	   ├──(惰性对账: 工单已完成而回调未留下痕迹)──> approved
//	   └──(审批驳回/取消/终止，或工单已取消/终止的惰性对账)──> rejected
//
// approved 是「已领取、执行在后台跑」的中间态：回调不在审批请求里跑 LLM，而是投递给有界
// 执行体（见 ProposalGateway.StartExecutionWorker）。投递走的是进程内队列，故进程在领取后、
// 写入 exec_run_id 前退出就会丢单，提案停在 approved。
//
// 这种停滞**不能指望工单侧重推**：回调返回「已受理」那一刻 flow 就把工单记为处理成功，
// 而 flow 的清扫/改期/取消执行全部只认「业务待处理」，FlowBizHandle 因此不会再被调用。
// 自愈靠 agentops 自己的停滞清扫（见 ProposalGateway.SweepStalledProposals），重投的幂等性
// 由执行运行的幂等键保证；治理看板以 StuckApproved 透出尚未被清扫收走的条数。
const (
	ProposalStatusPending  = "pending"  // 待审批（已登记，可能已发起审批工单）
	ProposalStatusApproved = "approved" // 审批通过（回调已领取，执行中）
	ProposalStatusRejected = "rejected" // 已驳回（审批未通过/取消/终止）
	ProposalStatusExecuted = "executed" // 已执行（执行档完成，含校验结论）
	ProposalStatusFailed   = "failed"   // 执行失败
)

// 提案风险级（与 ai/tools.propose_remediation 的取值契约一致）
const (
	ProposalRiskLow      = "low"
	ProposalRiskMedium   = "medium"
	ProposalRiskHigh     = "high"
	ProposalRiskCritical = "critical"
)

// IsProposalTerminal 提案是否处于终态（终态不再随审批状态流转）
func IsProposalTerminal(status string) bool {
	switch status {
	case ProposalStatusRejected, ProposalStatusExecuted, ProposalStatusFailed:
		return true
	default:
		return false
	}
}
