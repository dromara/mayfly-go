package application

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/imsg"
	aientity "mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/memory"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

// 本文件实现 RunService 的「人类反馈回流 + 离线评测回放」（计划阶段 4.3）：
//   - Feedback：对终态运行点赞/点踩 → 落库 + 经 ai memory 系统沉淀为操作员经验记忆，
//     形成学习闭环（后续会话经 memory 注入/检索召回）。
//   - ReplayEval：复制终态运行的原始输入用诊断档重跑一次，再由审查档对比新旧结论打分，
//     回归 Agent 诊断质量（可手动触发，历史事件重放语义）。

// evalJudgeInstruction 评测打分运行的 system 指令（reviewer 档）
const evalJudgeInstruction = `本次运行是「离线评测的打分环节」。对比「原运行结论」与「重放运行的新结论」：
1. 从准确性（结论与事实/指标是否一致）、完整性（关键风险是否覆盖）、可操作性（建议是否可执行）三个维度对比；
2. 最终必须输出一行 JSON：{"score": 0到10的整数, "verdict": "一句话判定"}；
3. 只读操作，不得执行任何变更。`

// evalScoreRe 从审查档输出中宽松提取评分 JSON（容错自然语言包裹）
var evalScoreRe = regexp.MustCompile(`\{[^{}]*"score"\s*:\s*(\d+)[^{}]*\}`)

// Feedback 人类反馈：校验 + 落库 + 沉淀经验记忆（fail-open，记忆失败不影响反馈落库）
func (r *runServiceAppImpl) Feedback(ctx context.Context, runId uint64, feedback, note string) error {
	if !entity.IsValidFeedback(feedback) {
		return errorx.NewBizI(ctx, imsg.ErrFeedbackValueInvalid)
	}
	// 备注限长守卫：列宽 size:1000，超长直接截断（前端 prompt 无硬限制，防原始 DB 错误透出）
	if rs := []rune(note); len(rs) > 1000 {
		note = string(rs[:1000])
	}
	run, err := r.GetById(runId)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrRunNotFound)
	}
	if !entity.IsTerminal(run.Status) {
		return errorx.NewBizI(ctx, imsg.ErrFeedbackRequiresTerminal)
	}
	// 备注与 owner 账号名同样收敛在 Extra（无独立列），因此写入走「读旧值 + 整列写回 extra」：
	// 直接 Set("feedbackNote", ...) 会打到不存在的列上，表现为接口 200 但备注从未落库。
	// SetExtraValue 对空串是删键，所以「清空备注」也能真落库（不会被当“未传”而留下旧值）
	run.SetExtraValue(entity.ExtraKeyFeedbackNote, note)
	if _, err := r.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().Set("feedback", feedback).Set("extra", run.Extra),
		model.NewCond().EqStrict("id", runId)); err != nil {
		return err
	}
	logx.InfofContext(ctx, "[agentops-run] feedback saved: runNo=%s, feedback=%s", run.RunNo, feedback)
	r.saveFeedbackMemory(ctx, run, feedback, note)
	return nil
}

// saveFeedbackMemory 把反馈沉淀为操作员的经验记忆（学习闭环）。
// 归属身份：优先当前操作员，缺省回退运行 owner（定时/事件运行无操作员时）。
func (r *runServiceAppImpl) saveFeedbackMemory(ctx context.Context, run *entity.Run, feedback, note string) {
	defer gox.Recover()
	store := memory.DefaultStore
	if store == nil {
		return // 单测/记忆系统未装配时静默跳过（fail-open）
	}
	userId := ""
	if la := contextx.GetLoginAccount(ctx); la != nil {
		userId = strconv.FormatUint(uint64(la.Id), 10)
	}
	if userId == "" {
		userId = strconv.FormatUint(run.OwnerAccountId, 10)
	}

	verdict := "结论准确/有帮助"
	if feedback == entity.FeedbackDown {
		verdict = "结论有误/无帮助"
	}
	if note == "" {
		note = "无备注"
	}
	// 备注必须进记忆正文：它是「结论哪里不对/为什么有用」的唯一人因信号，也是前端 prompt
	// 对用户明说的承诺（「反馈备注（可选），将随经验沉淀进记忆系统」）。只归一化不拼进去，
	// 等于存了个永远读不到的字段，且记忆检索（memory_search）拿不到任何可据以改进行为的理由
	content := fmt.Sprintf("运维 Agent 运行「%s」(runNo=%s) 被人工反馈为%s（%s）。反馈备注：%s。运行输入摘要：%s。运行结论摘要：%s。",
		run.Title, run.RunNo, feedback, verdict, truncateForMemory(note, 300), truncateForMemory(run.Input, 300), truncateForMemory(run.Summary, 500))

	now := time.Now()
	item := &memory.MemoryItem{
		// ID 留空：DB 版 memory 存储要求 ID 为 uint64（t_ai_memory.id 自增），
		// 空值走新增由存储层生成；曾误传 SortableUUID 致 ParseUint 失败、沉淀全部失败
		UserID:    userId,
		Type:      "experience",
		Content:   content,
		Tags:      []string{"agentops", "run-feedback", feedback},
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  map[string]string{"runId": strconv.FormatUint(run.Id, 10), "runNo": run.RunNo},
	}
	if err := store.Save(ctx, []*memory.MemoryItem{item}); err != nil {
		logx.WarnfContext(ctx, "[agentops-run] save feedback memory failed (fail-open): runNo=%s, err=%v", run.RunNo, err)
	}
}

// ReplayEval 离线评测回放：同步落库评测运行（running）→ 异步执行「诊断档重放 + 审查档打分」。
// 命中幂等键（同一原始运行同分钟内重复发起）时短路返回既有评测运行。
func (r *runServiceAppImpl) ReplayEval(ctx context.Context, runId uint64) (*entity.Run, error) {
	orig, err := r.GetById(runId)
	if err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrRunNotFound)
	}
	if !entity.IsTerminal(orig.Status) {
		return nil, errorx.NewBizI(ctx, imsg.ErrEvalRequiresTerminal)
	}
	if orig.Input == "" && orig.Instruction == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrEvalRequiresOriginalInput)
	}
	if orig.OwnerAccountId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrEvalNoOwner)
	}
	// 急停守护：评测一次烧两段 LLM，恰是开关拉起时最该拦的配额入口。必须在落库前判定
	// （与 orchestrator 同模式）：评测运行以 running 直接落库、由专属 goroutine 驱动，
	// 先落再拒会留下悬挂运行。RunSync 的豁免不适用于这里：那条豁免的理由是「已批准的变更
	// 必须能收尾」，评测是只读诊断，没有必须收尾的东西
	if err := guardDispatch(r.governance); err != nil {
		return nil, err
	}

	runNo := stringx.SortableUUID()
	// 分钟桶幂等：防双击重复发起；跨分钟允许再次评测（Agent/配置可能已演进）
	idemKey := evalTriggerRefPrefix + strconv.FormatUint(orig.Id, 10) + ":" + time.Now().Format("200601021504")

	now := time.Now()
	evalRun := &entity.Run{
		RunNo:          runNo,
		TriggerType:    entity.TriggerTypeManual,
		TriggerRef:     evalTriggerRefPrefix + strconv.FormatUint(orig.Id, 10),
		ProfileId:      headless.ProfileDiagnostician,
		Title:          orig.Title,
		RunRole:        entity.RunRoleEvalReplay,
		Status:         entity.RunStatusRunning,
		Instruction:    orig.Instruction,
		Input:          orig.Input,
		OwnerAccountId: orig.OwnerAccountId,
		OwnerUsername:  orig.OwnerUsername,
		TimeoutSeconds: orig.TimeoutSeconds,
		MaxStep:        orig.MaxStep,
		TokenBudget:    orig.TokenBudget,
		IdempotencyKey: idemKey,
		StartedAt:      &now,
		RunnerInstance: runInstanceId,
	}
	if err := r.InsertIfAbsent(ctx, evalRun, "idempotency_key"); err != nil {
		return nil, err
	}
	if evalRun.Id == 0 {
		existing, gerr := r.GetRepo().GetByIdempotencyKey(idemKey)
		if gerr != nil {
			return nil, gerr
		}
		logx.DebugfContext(ctx, "[agentops-eval] deduplicated by idempotency key: %s (runNo=%s)", idemKey, existing.RunNo)
		return existing, nil
	}

	// 异步执行（评测为两次 LLM 运行，不能阻塞 HTTP）；终态写库用独立链路 ctx。
	// panic 在调用侧接住并补写终态：评测运行以 running 直接落库、由专属 goroutine 驱动，
	// 没人收尾就永久停在 running（与编排父运行同一形态）
	go func() {
		taskCtx := contextx.WithTraceId(context.Background())
		var panicErr error
		func() {
			defer gox.Recover(func(e error) { panicErr = e })
			r.executeEval(taskCtx, evalRun, orig)
		}()
		if panicErr != nil {
			logx.ErrorfContext(taskCtx, "[agentops-eval] evaluation panicked (recovered): runNo=%s, err=%v", evalRun.RunNo, panicErr)
			r.finishEval(taskCtx, evalRun, entity.RunStatusFailed, evalUsage{}, "", "",
				fmt.Sprintf("evaluation panicked: %v", panicErr), nil)
		}
	}()
	return evalRun, nil
}

// executeEval 评测执行体：诊断档重放原始输入 → 审查档对比打分 → 回写评测运行终态
func (r *runServiceAppImpl) executeEval(ctx context.Context, evalRun *entity.Run, orig *entity.Run) {
	// panic 由调用侧统一接住并补写终态（见 ReplayEval）：在此只 recover 会把失败吞成「运行永久 running」

	// 运行级链路 ctx（终态写库用）；接入在途取消跟踪：急停/Cancel 取消 ctx 后两段 LLM
	// 运行快速收尾（未接入时 Cancel 仅置库态，goroutine 仍会把 LLM 烧完，违背急停承诺），
	// untrackInflight 顺带消费 Cancel 早于注册时残留的 cancelReq 标记，避免 map 泄漏
	runCtx, cancel := context.WithCancel(ctx)
	r.trackInflight(evalRun.Id, cancel)
	defer func() {
		r.untrackInflight(evalRun.Id)
		cancel()
	}()

	timeout := time.Duration(orig.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultRunTimeout
	}

	// 1) 诊断档重放（输入/指令/预算均快照自原始运行，保证可比性）
	res, runErr := headless.Run(runCtx, &headless.RunReq{
		ProfileId:      headless.ProfileDiagnostician,
		Instruction:    orig.Instruction,
		Input:          orig.Input,
		Title:          evalRun.Title,
		OwnerAccountId: orig.OwnerAccountId,
		OwnerUsername:  orig.OwnerUsername,
		Timeout:        timeout,
		MaxStep:        orig.MaxStep,
		MaxTotalTokens: orig.TokenBudget,
		Persist:        true,
		RunKind:        headless.RunKindManual,
		// 评测会话同为系统触发：标记 system 不进用户会话列表
		ConversationSource: aientity.ConversationSourceSystem,
	})
	if runErr != nil || res == nil || res.Status != headless.RunStatusSucceeded {
		r.finishEval(ctx, evalRun, entity.RunStatusFailed, evalUsage{},
			"", "", fmt.Sprintf("replay run failed: status=%s, err=%v", headlessStatusOf(res), runErr), res)
		return
	}

	// 2) 审查档打分（对比新旧结论）
	judgeRes, judgeErr := headless.Run(runCtx, &headless.RunReq{
		ProfileId:      headless.ProfileReviewer,
		Instruction:    evalJudgeInstruction,
		Input:          buildJudgeInput(orig, res.Output),
		Title:          orig.Title,
		OwnerAccountId: orig.OwnerAccountId,
		OwnerUsername:  orig.OwnerUsername,
		Timeout:        timeout,
		Persist:        true,
		RunKind:        headless.RunKindManual,
		// 同上：评测打分会话标记 system
		ConversationSource: aientity.ConversationSourceSystem,
	})
	judgeOutput := ""
	if judgeRes != nil {
		judgeOutput = judgeRes.Output
	}
	if judgeErr != nil {
		// 打分失败不影响重放事实：结论记新输出，错误附注
		r.finishEval(ctx, evalRun, entity.RunStatusSucceeded, usageOf(res, judgeRes),
			res.Output, "", "judge run failed: "+runErrMsg(judgeErr), res)
		return
	}

	// 3) 回写终态：得分 + 判定理由 + 重放新结论
	score, verdict := parseEvalScore(judgeOutput)
	r.finishEval(ctx, evalRun, entity.RunStatusSucceeded, usageOf(res, judgeRes),
		res.Output, judgeOutput, fmt.Sprintf("评测得分: %d/10。%s", score, verdict), res)
}

// finishEval 回写评测运行终态（仅当仍为 running；summary 组装：判定 + 新结论 + 附加信息）。
//
// replayRes 是诊断档重放的产出（可为 nil：失败/panic 路径）：它携带的会话/轮次必须随终态
// 回写——重放运行是 Persist:true 产生的持久化会话，不回写则前端「回放」按钮对每个评测运行
// 恒灰（按 conversationId 判可用），评分无法核查，评测会话成为库里无人可导航的孤儿记录
func (r *runServiceAppImpl) finishEval(ctx context.Context, evalRun *entity.Run, status string, usage evalUsage,
	replayOutput, judgeOutput, note string, replayRes *headless.RunResult) {
	now := time.Now()
	patch := base.NewPatch().
		Set("status", status).
		Set("finishedAt", now).
		Set("summary", buildEvalSummary(note, judgeOutput, replayOutput))
	if replayRes != nil && replayRes.ConversationId != 0 {
		patch = patch.Set("conversationId", replayRes.ConversationId)
	}
	if replayRes != nil && replayRes.TurnId != "" {
		patch = patch.Set("turnId", replayRes.TurnId)
	}
	if evalRun.StartedAt != nil {
		patch = patch.Set("costMs", now.Sub(*evalRun.StartedAt).Milliseconds())
	}
	if usage.total > 0 {
		// 输入/输出拆分必须一并回写：只写 total 会让看板的 prompt/completion 少计这一段，
		// 跑过评测后「输入 + 输出 ≠ 总数」，成本归因（上下文膨胀 vs 输出啰嗦）这个指标直接失真
		patch = patch.
			Set("promptTokens", usage.prompt).
			Set("completionTokens", usage.completion).
			Set("totalTokens", usage.total)
	}
	if status != entity.RunStatusSucceeded {
		patch = patch.Set("err", truncateRunErr(note))
	}
	rows, err := r.GetRepo().UpdateColsByCond(ctx, patch,
		model.NewCond().EqStrict("id", evalRun.Id).EqStrict("status", entity.RunStatusRunning))
	if err != nil {
		logx.ErrorfContext(ctx, "[agentops-eval] finish update failed: runNo=%s, err=%v", evalRun.RunNo, err)
		return
	}
	if rows != 1 {
		// 没抢到终态写入（已被取消/启动清扫收尾）：只记日志，不覆写他方结论
		logx.WarnfContext(ctx, "[agentops-eval] finish skipped, terminal state written elsewhere: runNo=%s, computedStatus=%s",
			evalRun.RunNo, status)
		return
	}
	logx.InfofContext(ctx, "[agentops-eval] finished: runNo=%s, status=%s", evalRun.RunNo, status)
}

// buildJudgeInput 构造打分输入：原始运行结论 + 重放新结论
func buildJudgeInput(orig *entity.Run, newOutput string) string {
	return "运维目标/输入：\n" + truncateForMemory(orig.Input, 1000) +
		"\n\n原运行结论：\n" + truncateForMemory(orig.Summary, 2000) +
		"\n\n重放运行的新结论：\n" + truncateForMemory(newOutput, 2000) +
		"\n\n请对比两份结论并输出评分 JSON。"
}

// buildEvalSummary 组装评测运行结论摘要
func buildEvalSummary(note, judgeOutput, replayOutput string) string {
	out := note
	if judgeOutput != "" {
		out += "\n\n--- 审查档判定输出 ---\n" + truncateForMemory(judgeOutput, 2000)
	}
	if replayOutput != "" {
		out += "\n\n--- 重放运行新结论 ---\n" + truncateForMemory(replayOutput, 3000)
	}
	return out
}

// parseEvalScore 从审查档输出宽松提取评分与判定语；未提取到时记 0 分并保留原输出
func parseEvalScore(output string) (int, string) {
	m := evalScoreRe.FindStringSubmatch(output)
	if len(m) >= 2 {
		if score, err := strconv.Atoi(m[1]); err == nil {
			if score > 10 {
				score = 10
			}
			return score, output
		}
	}
	return 0, output
}

// evalUsage 评测两段（重放 + 打分）运行的 token 用量汇总。
//
// 取三元组而非单一 total：入参本就是两个 headless.RunResult，输入/输出拆分是拿得到的，
// 只汇总成 total 就丢了（见 finishEval 的回写注释）。
type evalUsage struct{ prompt, completion, total int64 }

func usageOf(res, judgeRes *headless.RunResult) evalUsage {
	return evalUsage{
		prompt:     promptTokensOf(res) + promptTokensOf(judgeRes),
		completion: completionTokensOf(res) + completionTokensOf(judgeRes),
		total:      totalTokensOf(res) + totalTokensOf(judgeRes),
	}
}

// headlessStatusOf 取无头运行状态（nil 安全）
func headlessStatusOf(res *headless.RunResult) string {
	if res == nil {
		return "nil"
	}
	return res.Status
}

// runErrMsg 取运行错误信息（nil 安全）
func runErrMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// truncateForMemory 截断文本用于记忆/评测摘要（按 rune 截断，避免切断多字节字符）
func truncateForMemory(s string, max int) string {
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max]) + "...(截断)"
}
