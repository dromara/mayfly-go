package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	aitools "mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

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
