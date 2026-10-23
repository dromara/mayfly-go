package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

// Orchestrator 多 Agent 编排器（计划阶段 4.1，Supervisor 模式）：
//
//	Planner（诊断档）把运维目标拆解为有序步骤清单
//	  → Executor（只读步骤直跑诊断档；涉及变更的步骤经 propose_remediation 登记提案转人工审批）
//	  → Reviewer（审查档）综合各步骤结果判定目标是否达成并给出最终报告。
//
// 每个子 Agent = 一次独立 Run（trigger_ref=orch:<父RunId>:step:<n>），编排轨迹天然落在
// 执行历史中（回放/审计/成本统计零成本复用，对齐计划「复用 ChannelLifecycle 记录编排轨迹」意图）。
// 父 Run 仅作编排容器（承载最终报告与编排状态），不经工作池执行，由编排 goroutine 驱动。
type Orchestrator interface {
	// Orchestrate 发起一次编排：落库父运行（running）并异步执行编排流程，立即返回父运行。
	// 编排进度经执行历史中 trigger_ref=orch:<id>:* 的子运行与父运行 summary 观察。
	Orchestrate(ctx context.Context, req *OrchestrateReq) (*entity.Run, error)
}

// maxOrchestrationSteps 单次编排最大步骤数（防 Planner 产出无界计划）
const maxOrchestrationSteps = 10

// orchTriggerRefPrefix 编排子运行的 TriggerRef 前缀（形态 `orch:<父RunId>:step:<n>`）。
//
// 必须用常量而不是在拼接处重写字面量：与 evalTriggerRefPrefix 同一个理由——该前缀同时参与
// 幂等键与运行归属识别，漂移到哪一处都只是文本不同，表现为「子运行找不到父运行」或
// 「租约按单步算而腰斩长运行」，编译与界面都不红。父运行的 TriggerRef 直接取
// entity.TriggerTypeOrchestration（与子前缀无派生关系，但同属一个枚举真源）。
const orchTriggerRefPrefix = "orch:"

// maxConcurrentOrchestrations 本实例并发编排上限（每次编排串行多次 LLM 调用，须限流）
const maxConcurrentOrchestrations = 2

// OrchestrateReq 编排请求
type OrchestrateReq struct {
	// Goal 运维目标（必填），如「排查 db-1 CPU 持续偏高的根因并给出处置建议」
	Goal string
	// Instruction 额外的编排要求（可空；与各档位静态指令叠加）
	Instruction string
	// OwnerAccountId 编排运行身份账号 ID（必填；fail-closed，工具权限按此收敛）
	OwnerAccountId uint64
	// OwnerUsername 编排运行身份账号名
	OwnerUsername string
	// TimeoutSeconds 单步运行超时秒数（0=用服务默认）
	TimeoutSeconds int
	// MaxStep 单步运行最大步数（0=用 Profile 默认）
	MaxStep int
	// TokenBudget 单步运行 token 预算熔断（0=不熔断）
	TokenBudget int64
}

// orchestratorAppImpl 编排器实现
type orchestratorAppImpl struct {
	runService RunService `inject:"T"`
	governance Governance `inject:"T"`
	// proposalRepo 反查「各步骤实际登记了哪些提案」：编排报告的提案清单必须由库决定，
	// 不能采信步骤自述（理由见 proposalsFactText）
	proposalRepo repository.Proposal `inject:"T"`

	// sem 本实例并发编排信号量（Orchestrate 立即返回、执行异步，须以信号量限流）。
	// semOnce 保证并发首调只初始化一次：无同步的懒初始化是数据竞态，且后到的空 channel
	// 会覆盖已计数的信号量（限流被重置绕过）甚至导致旧 goroutine 释放时永久阻塞
	sem     chan struct{}
	semOnce sync.Once
}

var _ Orchestrator = (*orchestratorAppImpl)(nil)

// Orchestrate 发起编排：校验 → 限流 → 落库父运行（running）→ 异步执行编排流程
func (o *orchestratorAppImpl) Orchestrate(ctx context.Context, req *OrchestrateReq) (*entity.Run, error) {
	if req == nil || strings.TrimSpace(req.Goal) == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrOrchGoalRequired)
	}
	if req.OwnerAccountId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrOrchOwnerRequired)
	}
	// 急停守护：编排会派生多次运行，开关拉起时拒绝发起（fail-closed 透出给前端）。
	// 必须在落库父运行之前判定：父运行由专属 goroutine 驱动，先落 running 再拒绝会留下悬挂运行
	if err := guardDispatch(o.governance); err != nil {
		return nil, err
	}

	// 并发限流：非阻塞获取信号量，满则拒绝（编排是长流程，排队无意义）
	o.semOnce.Do(func() {
		o.sem = make(chan struct{}, maxConcurrentOrchestrations)
	})
	select {
	case o.sem <- struct{}{}:
	default:
		return nil, errorx.NewBizI(ctx, imsg.ErrOrchConcurrencyLimit, "limit", maxConcurrentOrchestrations)
	}

	goal := strings.TrimSpace(req.Goal)
	now := time.Now()
	parent := &entity.Run{
		RunNo:          stringx.SortableUUID(),
		TriggerType:    entity.TriggerTypeOrchestration,
		TriggerRef:     entity.TriggerTypeOrchestration,
		ProfileId:      headless.ProfileDiagnostician,
		Title:          truncateForMemory(goal, 100),
		RunRole:        entity.RunRoleOrchParent,
		Status:         entity.RunStatusRunning,
		Input:          goal,
		OwnerAccountId: req.OwnerAccountId,
		OwnerUsername:  req.OwnerUsername,
		TimeoutSeconds: req.TimeoutSeconds,
		MaxStep:        req.MaxStep,
		TokenBudget:    req.TokenBudget,
		IdempotencyKey: "auto:" + stringx.SortableUUID(), // 编排容器不去重（每次编排都是新流程）
		StartedAt:      &now,
	}
	// 直接落库 running（不经工作池：编排由专属 goroutine 驱动，子运行才经 RunService 执行）
	if err := o.runService.Save(ctx, parent); err != nil {
		<-o.sem
		return nil, err
	}

	go func() {
		defer func() { <-o.sem }()
		// panic 不能只 recover 了事：父运行是直接 Save(running) 并由本 goroutine 驱动（不经工作池，
		// 工作池兜底轮询只重驱 queued），没人收尾就会永久停在 running 并虚占治理看板的「运行中」
		taskCtx := contextx.WithTraceId(context.Background())
		var panicErr error
		func() {
			defer gox.Recover(func(e error) { panicErr = e })
			o.run(taskCtx, parent, req)
		}()
		if panicErr != nil {
			logx.ErrorfContext(taskCtx, "[agentops-orch] orchestration panicked (recovered): runNo=%s, err=%v", parent.RunNo, panicErr)
			o.finishParent(taskCtx, parent, entity.RunStatusFailed, "orchestration panicked: "+panicErr.Error(), orchUsage{})
		}
	}()

	logx.InfofContext(ctx, "[agentops-orch] orchestration started: runNo=%s, goal=%s", parent.RunNo, goal)
	return parent, nil
}

// orchUsage 一次编排的成本累计。
//
// 父运行必须反映整棵编排树的真实消耗：旧实现不汇总，导致执行历史里父运行 Tokens=0
// 而子运行合计达十万级，治理看板 token KPI 与「这次编排花了多少」全部失真（实测）。
type orchUsage struct {
	prompt     int64
	completion int64
	total      int64
}

// add 累加一条子/阶段运行的用量（nil 安全）
func (u *orchUsage) add(run *entity.Run) {
	if run == nil {
		return
	}
	u.prompt += run.PromptTokens
	u.completion += run.CompletionTokens
	u.total += run.TotalTokens
}

// run 编排执行体：Planner → 步骤执行 → Reviewer → 回写父运行终态
func (o *orchestratorAppImpl) run(ctx context.Context, parent *entity.Run, req *OrchestrateReq) {
	parentRef := orchTriggerRefPrefix + strconv.FormatUint(parent.Id, 10)
	var usage orchUsage

	// 1) Planner：拆解目标为步骤清单
	planRes, planErr := o.runService.RunSync(ctx, &EnqueueReq{
		TriggerType:    entity.TriggerTypeOrchestration,
		TriggerRef:     parentRef + ":planner",
		ParentRunId:    parent.Id,
		ProfileId:      headless.ProfileDiagnostician,
		Title:          truncateForMemory(req.Goal, 80),
		RunRole:        entity.RunRoleOrchPlan,
		Instruction:    strings.TrimSpace(orchPlannerInstruction + "\n\n" + req.Instruction),
		Input:          req.Goal,
		OwnerAccountId: req.OwnerAccountId,
		OwnerUsername:  req.OwnerUsername,
		TimeoutSeconds: req.TimeoutSeconds,
		MaxStep:        req.MaxStep,
		TokenBudget:    req.TokenBudget,
		IdempotencyKey: parentRef + ":planner",
	})
	if planErr != nil || planRes == nil || planRes.Status != entity.RunStatusSucceeded {
		o.finishParent(ctx, parent, entity.RunStatusFailed, "planner run failed: "+runErrMsg(planErr), usage)
		return
	}
	usage.add(planRes)

	// 2) 步骤执行：解析计划 → 逐步骤派生运行（写步骤转提案，经审批链路受控执行）
	steps := parsePlanSteps(planRes.Summary)
	if len(steps) == 0 {
		// 计划解析失败：把 Planner 输出整体视为单只读步骤（宽进，编排不因格式失真中断）
		steps = []*orchStep{{Title: "执行目标", Action: req.Goal, ReadOnly: true}}
	}
	if len(steps) > maxOrchestrationSteps {
		steps = steps[:maxOrchestrationSteps]
	}

	var report strings.Builder
	fmt.Fprintf(&report, "## 编排目标\n%s\n\n## 步骤执行结果\n", req.Goal)
	var stepSummaries strings.Builder
	// stepRunIds 汇总已派生的步骤运行 id，供事后反查它们实际登记的提案
	stepRunIds := make([]uint64, 0, len(steps))
	for i, step := range steps {
		// 急停守护：开关拉起时停止派发后续步骤（已完成的步骤结论保留）
		if err := guardDispatch(o.governance); err != nil {
			fmt.Fprintf(&report, "- 步骤 %d 起因派发守护而中止：%s\n", i+1, err.Error())
			o.finishParent(ctx, parent, entity.RunStatusCanceled, report.String(), usage)
			return
		}
		// 父运行状态复查：读失败与「已结束」是两回事，必须分开处理。
		// 旧写法把三个条件合并成一个 return：任何一次瞬时 DB 抖动（连接池、超时）都会被当成
		// 「父运行已结束」而不收尾，编排静默中止且父运行永久挂在 running
		cur, err := o.runService.GetById(parent.Id)
		if err != nil {
			logx.ErrorfContext(ctx, "[agentops-orch] parent state reload failed, abort orchestration: runNo=%s, err=%v", parent.RunNo, err)
			o.finishParent(ctx, parent, entity.RunStatusFailed,
				report.String()+"\n## 编排中止\n读取父运行状态失败："+err.Error(), usage)
			return
		}
		if cur == nil || cur.Status != entity.RunStatusRunning {
			// 用户已单独取消本编排（Cancel 仅置库态）：停止派发后续步骤、不覆写已落库的终态
			lastStatus := "nil"
			if cur != nil {
				lastStatus = cur.Status
			}
			logx.InfofContext(ctx, "[agentops-orch] parent no longer running, stop dispatch: runNo=%s, status=%s",
				parent.RunNo, lastStatus)
			return
		}
		run, err := o.runStep(ctx, parent, parentRef, i+1, step, req)
		usage.add(run)
		if run != nil {
			stepRunIds = append(stepRunIds, run.Id)
		}
		outcome := "（运行未产出结论）"
		if err != nil {
			outcome = "（运行失败：" + runErrMsg(err) + "）"
		} else if run != nil && run.Summary != "" {
			outcome = run.Summary
		}
		fmt.Fprintf(&report, "- 步骤 %d【%s】：%s\n", i+1, step.Title, outcome)
		fmt.Fprintf(&stepSummaries, "- 【%s】%s\n", step.Title, outcome)
	}

	// 3) Reviewer：综合各步骤结果判定目标达成情况
	//
	// 提案事实清单同时追参加父报告与 Reviewer 输入：前者是给运维看的终态，后者是模型判定
	// 「目标是否达成」的依据，两者必须用同一份库里的结果而不是各算一套
	proposalsFact := o.proposalsFactText(ctx, stepRunIds)
	report.WriteString(proposalsFact)
	stepSummaries.WriteString(proposalsFact)
	reviewRes, reviewErr := o.runService.RunSync(ctx, &EnqueueReq{
		TriggerType:    entity.TriggerTypeOrchestration,
		TriggerRef:     parentRef + ":reviewer",
		ParentRunId:    parent.Id,
		ProfileId:      headless.ProfileReviewer,
		Title:          truncateForMemory(req.Goal, 80),
		RunRole:        entity.RunRoleOrchReview,
		Instruction:    orchReviewerInstruction,
		Input:          "编排目标：\n" + req.Goal + "\n\n各步骤执行结果：\n" + stepSummaries.String(),
		OwnerAccountId: req.OwnerAccountId,
		OwnerUsername:  req.OwnerUsername,
		TimeoutSeconds: req.TimeoutSeconds,
		MaxStep:        req.MaxStep,
		TokenBudget:    req.TokenBudget,
		IdempotencyKey: parentRef + ":reviewer",
	})
	if reviewErr != nil || reviewRes == nil || reviewRes.Status != entity.RunStatusSucceeded {
		fmt.Fprintf(&report, "\n## 审查环节失败\n%s", runErrMsg(reviewErr))
		o.finishParent(ctx, parent, entity.RunStatusFailed, report.String(), usage)
		return
	}

	usage.add(reviewRes)

	// 4) 回写父运行终态：最终报告（成功）
	fmt.Fprintf(&report, "\n## 最终结论（Reviewer）\n%s", reviewRes.Summary)
	o.finishParent(ctx, parent, entity.RunStatusSucceeded, report.String(), usage)
}

// runStep 执行单个编排步骤：只读步骤直跑诊断档；涉及变更的步骤仍用诊断档执行，
// 但指令限定「仅经 propose_remediation 登记变更提案」，由提案→审批→受控执行链路落地。
func (o *orchestratorAppImpl) runStep(ctx context.Context, parent *entity.Run, parentRef string, idx int, step *orchStep, req *OrchestrateReq) (*entity.Run, error) {
	instruction := orchStepReadonlyInstruction
	if !step.ReadOnly {
		instruction = orchStepProposalInstruction
	}
	return o.runService.RunSync(ctx, &EnqueueReq{
		TriggerType:    entity.TriggerTypeOrchestration,
		TriggerRef:     parentRef + ":step:" + strconv.Itoa(idx),
		ParentRunId:    parent.Id,
		ProfileId:      headless.ProfileDiagnostician,
		Title:          truncateForMemory(step.Title, 80),
		RunRole:        entity.RunRoleOrchStep,
		Instruction:    instruction,
		Input:          "编排目标：\n" + req.Goal + "\n\n本步骤任务：\n" + step.Action,
		OwnerAccountId: req.OwnerAccountId,
		OwnerUsername:  req.OwnerUsername,
		TimeoutSeconds: req.TimeoutSeconds,
		MaxStep:        req.MaxStep,
		TokenBudget:    req.TokenBudget,
		IdempotencyKey: parentRef + ":step:" + strconv.Itoa(idx),
	})
}

// proposalsFactHeading 提案事实清单的小标题（父报告与 Reviewer 输入共用一份文案，不两处各写一遍）
const proposalsFactHeading = "\n## 本次编排实际登记的变更提案（以此清单为准）\n"

// proposalsFactText 按步骤运行 id 反查「确实登记了哪些变更提案」，产出供报告与 Reviewer 引用的事实清单。
//
// 存在理由（真机实测）：某个只读步骤在结论里写「并为该探测动作登记变更提案」，而那一轮
// 根本没调过 propose_remediation；编排把各步骤自述原文喂给 Reviewer，最终报告就列了一条
// 并不存在的待审提案。审批人照报告去提案列表找不到它，只会怀疑「是列表漏了」—— 谎报比不报更坏。
// 「登记了什么」是库里查得到的事实，必须由它决定，不能由模型散文决定。
//
// 反查失败时不静默省略：那会让报告退回「只有步骤自述」的旧形态，恰是本次要修的洞；
// 改为明确写出「未经核实」，让读报告的人知道这份清单缺失
func (o *orchestratorAppImpl) proposalsFactText(ctx context.Context, stepRunIds []uint64) string {
	var sb strings.Builder
	sb.WriteString(proposalsFactHeading)
	if len(stepRunIds) == 0 {
		sb.WriteString("本次编排未产出任何步骤运行，因此确定没有登记变更提案。\n")
		return sb.String()
	}

	proposals, err := o.proposalRepo.SelectByCond(
		model.NewCond().In("runId", stepRunIds).OrderBy("id asc"),
		"id", "title", "action", "risk", "status", "runId")
	if err != nil {
		logx.ErrorfContext(ctx, "[agentops-orch] list registered proposals failed, report keeps unverified step claims: %v", err)
		sb.WriteString("提案清单查询失败：" + err.Error() + "。上述步骤自述中「已登记提案」的说法未经核实。\n")
		return sb.String()
	}
	if len(proposals) == 0 {
		sb.WriteString("0 条。任何步骤结论里声称已登记提案的表述均不成立，不得据此认定存在待审事项。\n")
		return sb.String()
	}
	for _, proposal := range proposals {
		fmt.Fprintf(&sb, "- 提案 #%d【%s】动作=%s 风险=%s 状态=%s（来源运行 %d）\n",
			proposal.Id, proposal.Title, proposal.Action, proposal.Risk, proposal.Status, proposal.RunId)
	}
	return sb.String()
}

// finishParent 回写父运行终态（仅当仍为 running，summary 承载最终编排报告）
func (o *orchestratorAppImpl) finishParent(ctx context.Context, parent *entity.Run, status, summary string, usage orchUsage) {
	now := time.Now()
	patch := base.NewPatch().
		Set("status", status).
		Set("summary", summary).
		Set("finishedAt", now).
		Set("promptTokens", usage.prompt).
		Set("completionTokens", usage.completion).
		Set("totalTokens", usage.total)
	if parent.StartedAt != nil {
		patch = patch.Set("costMs", now.Sub(*parent.StartedAt).Milliseconds())
	}
	if status != entity.RunStatusSucceeded {
		patch = patch.Set("err", truncateRunErr(summary))
	}
	if err := o.runService.UpdateColsByCond(ctx, patch,
		model.NewCond().EqStrict("id", parent.Id).EqStrict("status", entity.RunStatusRunning)); err != nil {
		logx.ErrorfContext(ctx, "[agentops-orch] finish parent failed: runNo=%s, err=%v", parent.RunNo, err)
		return
	}
	logx.InfofContext(ctx, "[agentops-orch] orchestration finished: runNo=%s, status=%s, totalTokens=%d",
		parent.RunNo, status, usage.total)
}

// orchStep 编排步骤（Planner 产出的 JSON 项）
type orchStep struct {
	Title    string `json:"title"`
	Action   string `json:"action"`
	ReadOnly bool   `json:"readonly"`
}

// parsePlanSteps 宽松解析 Planner 输出中的步骤 JSON 数组：
// 截取首个 '[' 到最后一个 ']' 之间的内容反序列化；解析失败返回空（调用方降级为单步骤）。
func parsePlanSteps(output string) []*orchStep {
	start := strings.Index(output, "[")
	end := strings.LastIndex(output, "]")
	if start < 0 || end <= start {
		return nil
	}
	var steps []*orchStep
	if err := json.Unmarshal([]byte(output[start:end+1]), &steps); err != nil {
		return nil
	}
	valid := make([]*orchStep, 0, len(steps))
	for _, s := range steps {
		if s == nil {
			continue
		}
		if strings.TrimSpace(s.Title) == "" {
			s.Title = truncateForMemory(s.Action, 40)
		}
		if strings.TrimSpace(s.Action) != "" {
			valid = append(valid, s)
		}
	}
	return valid
}

// 编排各档位的任务级 system 指令（与 Profile 静态指令叠加，强调「参数锁定」与「只读复查」）
const (
	orchPlannerInstruction = `本次运行是「多 Agent 编排的规划阶段」。请把运维目标拆解为有序步骤清单：
1. 仅输出 JSON 数组，格式：[{"title":"步骤标题","action":"该步骤具体做什么（含要查的资源/指标/判据）","readonly":true}]；
2. readonly=true 表示只读调查步骤（查指标/日志/配置）；涉及任何变更的步骤必须 readonly=false；
3. 变更类步骤只规划「要做什么变更、为什么」，不执行变更（将由提案审批链路受控执行）；
4. 步骤数不超过 10 个，每步应可独立执行并产出明确结论。`

	orchStepReadonlyInstruction = `本次运行是「多 Agent 编排的只读调查步骤」。仅使用只读工具完成步骤任务并产出结构化结论（事实 + 数据 + 判断）。不得执行任何变更；也不得声称自己已登记变更提案——登记提案不是只读调查步骤能做的事，是否存在待审提案统一由后续审查阶段的提案清单列定。`

	orchStepProposalInstruction = `本次运行是「多 Agent 编排的变更规划步骤」。该步骤涉及变更：
1. 请使用 propose_remediation 工具登记结构化变更提案（锁定参数 + 理由 + 风险级），严禁直接执行任何变更；
2. 若调查后发现无需变更或前提不成立，可不登记提案，直接说明依据；产出结构化结论。`

	orchReviewerInstruction = `本次运行是「多 Agent 编排的审查阶段」。综合各步骤执行结果：
1. 判定编排目标是否达成（达成/部分达成/未达成）；
2. 已登记未审批的变更提案以下文「本次编排实际登记的变更提案」清单为准：清单未列出的提案就是不存在，不得从步骤自述文本里推断并写入结论；
3. 给出后续建议（如需补充调查、需人工介入、需回滚）。不得执行任何变更。`
)
