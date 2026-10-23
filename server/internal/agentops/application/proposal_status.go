package application

import (
	"context"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

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
