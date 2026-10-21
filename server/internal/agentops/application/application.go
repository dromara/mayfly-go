package application

import (
	"context"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

// mergeExtra 更新保存时把库中现值 Extra 里「本次没带的键」继承受新实体。
//
// 为什么必需：展示型字段（owner 账号名/备注）收敛进 Extra 后，extra 整列写回会覆盖整个 JSON；
// 若不先继承旧值，任何不经过本次表单/请求的键（例如运行行的 allowedToolNames、未来新增的扩展键）
// 会被静默抹掉。已存在的键以新值为准（本次编辑的意图优先）。
func mergeExtra(dst *model.ExtraData, src model.ExtraData) {
	if src.Extra == nil {
		return
	}
	for k, v := range src.Extra {
		if _, ok := dst.Extra[k]; ok {
			continue
		}
		if dst.Extra == nil {
			dst.Extra = collx.M{}
		}
		dst.Extra[k] = v
	}
}

// saveColumns 组合「业务可编辑列 + 审计列」，供显式列更新使用。
//
// 显式列更新必须带上审计列，否则 modifier/modifier_id 不随 Select 写入，修改人会停在创建者
// （见 entity.AuditUpdateColumns 注释）。返回副本而非 append 实体包的包级切片，避免写入共享底层数组。
func saveColumns(editable []string) []string {
	out := make([]string, 0, len(editable)+len(entity.AuditUpdateColumns))
	out = append(out, editable...)
	return append(out, entity.AuditUpdateColumns...)
}

// InitIoc 注册 agentops 应用服务
func InitIoc() {
	ioc.Register(new(runServiceAppImpl))
	ioc.Register(new(agentTaskAppImpl))
	ioc.Register(new(triggerServiceAppImpl))
	ioc.Register(new(proposalGatewayImpl))
	ioc.Register(new(governanceAppImpl))
	ioc.Register(new(orchestratorAppImpl))
}

// Init 应用初始化（在 DB 迁移与 ai 运行时装配之后由 starter 调用）：
// 先接线提案网关（flow 审批回调 + ai 登记端口），再收敛一批重启遗留的 running 运行，
// 然后启动 Run 有界工作池（附带周期租约清扫），最后扫描绑定定时任务 cron 并处理 misfire。
// 接线必须先于工作池启动（见 registerProposalIntegration）；启动清扫先于 Start，
// 使治理看板与重叠防护从干净状态开始（残留判定按租约过期，见 SweepInterruptedRuns）。
func Init() {
	registerProposalIntegration()
	sweepCtx := contextx.WithTraceId(context.Background())
	if err := GetRunService().SweepInterruptedRuns(sweepCtx); err != nil {
		logx.Errorf("[agentops] sweep interrupted runs failed: %v", err)
	}
	// 提案停滞清扫也先跑一轮：受控执行体投递的是进程内队列，上一个进程在 worker 取出前
	// 退出就丢了单，而 flow 侧的恢复入口对这类工单已全部不可达（详见 SweepStalledProposals）。
	// 必须排在 registerProposalIntegration 之后：重投需要执行体已启动，否则只会报未启动。
	// 单独套一层时限：投递是阻塞式的（队列满时等执行体空出名额），而这一轮清扫在**工作池启动与
	// cron 绑定之前**，被一条投递拖住就会变成「进程已起、HTTP 可用，但什么都不调度」
	sweepStalledCtx, cancelStalledSweep := context.WithTimeout(sweepCtx, 5*time.Second)
	requeued, err := GetProposalGateway().SweepStalledProposals(sweepStalledCtx)
	cancelStalledSweep()
	if err != nil {
		logx.Errorf("[agentops] sweep stalled proposals failed: %v", err)
	} else if requeued > 0 {
		logx.Warnf("[agentops] requeued %d stalled approved proposal(s) at startup", requeued)
	}
	GetRunService().Start()
	GetAgentTaskService().Init()
}

// Terminate 应用终止（进程退出时由 starter 的停机钩子调用，见 internal/agentops/init）：
// 先停提案受控执行异步体（它可能在收尾时再 Enqueue），再停 Run 工作池，
// 两者都等待在途任务自己写下终态。
//
// 为什么必需：不接线时 SIGTERM 直接抽干 goroutine，库里留下永久 running 的残留行，
// 而它会把同一任务的后续派发永久判为「上一次还在跑」而跳过（见 run_service.SweepInterruptedRuns）。
// 强杀（SIGKILL/OOM）下本钩子不会执行，那种残留由周期租约清扫兼底收敛。
func Terminate() {
	GetProposalGateway().StopExecutionWorker()
	GetRunService().Stop()
}

// GetRunService 运行聚合应用服务
func GetRunService() RunService {
	return ioc.Get[RunService]()
}

// GetAgentTaskService 定时计划任务应用服务
func GetAgentTaskService() AgentTaskService {
	return ioc.Get[AgentTaskService]()
}

// GetTriggerService 事件驱动触发器应用服务
func GetTriggerService() TriggerService {
	return ioc.Get[TriggerService]()
}

// GetProposalGateway 变更提案网关（自主运维闭环：提案 → 审批 → 受控执行 → 验证）。
// 同时是 ai/tools.ProposalRecorder 与 flow.FlowBizHandler 的实现方，由 init 期分别注入 flow 与 ai。
func GetProposalGateway() ProposalGateway {
	return ioc.Get[ProposalGateway]()
}

// GetGovernance 治理应用服务（急停开关 Kill Switch + 治理统计看板）
func GetGovernance() Governance {
	return ioc.Get[Governance]()
}

// GetOrchestrator 多 Agent 编排器（Planner → 步骤执行 → Reviewer，Supervisor 模式）
func GetOrchestrator() Orchestrator {
	return ioc.Get[Orchestrator]()
}
