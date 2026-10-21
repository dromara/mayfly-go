package application

import (
	"context"
	"errors"
	"fmt"

	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/application/dto"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"

	"github.com/cloudwego/eino/schema"
)

// RunHeadless 无头运行 Agent（定时任务/事件触发/审批/编排等统一入口）
//
// headless.Runner 端口的适配器实现（ai/application.Init 期注册到 headless 契约包，
// 供 flow/application 等消费方经 headless.Run 调用，破除循环依赖）。
//
// 从 flow/application/node_aitask.go 的无头调用雏形抽取泛化而来，与 WS 对话
// 共用 Agent 运行时。核心保障：
//   - 服务身份注入：运行前 contextx.WithLoginAccount(owner)，工具权限过滤按 owner 收敛；
//     owner 缺省 fail-closed 拒绝运行（防越权）。
//   - 一次性工具许可：req.AllowedToolNames 非空时注入 run grant，受控执行类运行只能调用清单内工具。
//   - 持久化复用 TurnRecorder：publisher 传 nil（仅收集不发布），天然获得 TurnItem
//     落库 → 审计/回放零成本复用；不落库时（Persist=false）为无痕运行。
//   - 超时/取消：Timeout>0 以 ctx 超时约束，终态细分为 timeout/canceled。
func RunHeadless(ctx context.Context, req *headless.RunReq) (*headless.RunResult, error) {
	if req == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrHeadlessReqNil)
	}
	// fail-closed：无服务身份拒绝运行（工具权限过滤依赖登录账号，缺省会越权/降级异常）
	if req.OwnerAccountId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrHeadlessOwnerRequired)
	}

	profileId := req.ProfileId
	if profileId == "" {
		profileId = headless.ProfileDiagnostician
	}
	ag, err := GetProfileAgentWithMaxStep(ctx, profileId, req.MaxStep)
	if err != nil {
		return nil, err
	}

	// 运行上下文：链路 ID + 服务身份注入（权限过滤按 owner 收敛）+ 超时约束
	runCtx := contextx.WithTraceId(ctx)
	runCtx = contextx.WithLoginAccount(runCtx, &model.LoginAccount{
		Id:       req.OwnerAccountId,
		Username: req.OwnerUsername,
	})
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, req.Timeout)
		defer cancel()
	}
	// 一次性工具许可：把「本次运行只能调用这些工具」下发到运行上下文，由护栏在调用链强制执行。
	// 这里不解释它的业务来源（审批过的提案 action / 编排步骤限定），仅透传，ai 与业务层仍无依赖
	if len(req.AllowedToolNames) > 0 {
		runCtx = tools.WithRunGrant(runCtx, &tools.RunGrant{AllowedToolNames: req.AllowedToolNames, ApprovedTools: req.ApprovedTools})
	}
	// 无人值守：本路径 publisher 传 nil，没有任何前端能回答中断卡片。必须显式告知工具层，
	// 否则「参数补全」这类等人填值的中断会把整轮挂死在一个永不应答的卡片上（运行直接 failed、
	// token 已花完却零产出）；工具体因此改为一把可重试的工具错误回传给模型自己补参。
	// 只影响「等人填参数」，不影响审批闸：无头运行不得做变更仍是底线（已批准的变更走 RunGrant）
	runCtx = tools.WithUnattended(runCtx)

	turnId := req.TurnId
	if turnId == "" {
		turnId = stringx.SortableUUID()
	}

	// 消息装配：任务级 system 提示词（可选）+ user 输入
	msgs := make([]*session.Message, 0, 2)
	if req.Instruction != "" {
		msgs = append(msgs, &session.Message{Role: schema.System, Content: req.Instruction})
	}
	msgs = append(msgs, &session.Message{Role: schema.User, Content: req.Input})

	runOpts := []agent.RunOption{agent.WithTurnId(turnId)}
	// 单次运行 token 预算熔断（>0 时生效）：防失控多步循环持续消耗 LLM 配额
	if req.MaxTotalTokens > 0 {
		runOpts = append(runOpts, agent.WithTokenBudget(req.MaxTotalTokens))
	}

	// 持久化接线：新建/复用会话 + TurnRecorder（publisher=nil，仅收集落库不推送）
	var (
		convId   uint64
		recorder *TurnRecorder
		convApp  Conversation
	)
	if req.Persist {
		convApp = GetConversationApp()
		convId = req.ConversationId
		if convId == 0 {
			conv, createErr := convApp.CreateConversation(runCtx, &dto.CreateConversationReq{
				// 标题直接透传：空值由会话应用层兜底成默认哨兵（见 headless.DefaultConversationTitle）。
				// 不在这里按 RunKind 编中文句子：那会把展示文案一次性写进库，
				// 英文界面永远看到中文、新增 RunKind 还要回来改 case
				Title:  req.Title,
				Source: req.ConversationSource,
			})
			if createErr != nil {
				return nil, createErr
			}
			convId = conv.Id
		}
		runOpts = append(runOpts, agent.WithRunSessionKey(fmt.Sprintf("conv:%d", convId)))
		recorder = NewTurnRecorder(GetTurnItemApp(), convId, turnId, nil)
		runOpts = append(runOpts,
			agent.WithOnChunk(recorder.OnChunk),
			agent.WithOnEvent(recorder.OnEvent),
		)
		// 用户输入落为 message TurnItem（会话回放可见"问了什么"；system 提示词不落库，
		// 与 WS 路径口径一致）
		userItem := protocol.NewMessageTurnItem(stringx.SortableUUID(), string(schema.User),
			[]protocol.ContentSegment{protocol.NewInputTextSegment(req.Input)})
		recorder.CollectProtocolItem(turnId, userItem, entity.ItemStatusSuccess)
	}

	// 业务关联回填：把解析出的会话 id 并入消费方（agentops）注入的关联标识，供运行期工具
	// （如 propose_remediation）把产物回链到所属 Run/会话；ai 运行时不解释其业务语义，仅透传。
	runAssoc := &headless.RunAssoc{ConversationId: convId}
	if prev := headless.RunAssocFrom(ctx); prev != nil {
		runAssoc.BizRunId = prev.BizRunId
	}
	runCtx = headless.WithRunAssoc(runCtx, runAssoc)

	res, runErr := ag.Run(runCtx, msgs, runOpts...)

	// 终态细分（超时/取消依赖 ctx.Err()，须在切换无取消 ctx 前判定）
	status := resolveRunStatus(runCtx, runErr)
	// 工具级失败随结果一并上报（run 本身可以成功，但变更其实没发生）
	var toolErrors []string
	if res != nil {
		toolErrors = res.ToolErrors
		// 中断挂起不能算成功：resolveRunStatus 只看 error，而中断是 agent 正常挂起（runErr 为 nil）。
		// 无头运行里没有审批应答者，这一步不定下来，「挂在审批上」就会被下游记成「已执行」
		if res.Interrupted && status == headless.RunStatusSucceeded {
			status = headless.RunStatusInterrupted
		}
	}

	// 收尾持久化：无取消 ctx（超时/取消后仍保证落库）
	if recorder != nil {
		saveCtx := context.WithoutCancel(runCtx)
		recorder.FinishStreaming()
		recorder.Flush(saveCtx)
		if res != nil && res.Usage != nil && convApp != nil {
			if incrErr := convApp.IncrTokenUsage(saveCtx, convId,
				int64(res.Usage.PromptTokens), int64(res.Usage.CompletionTokens), int64(res.Usage.TotalTokens)); incrErr != nil {
				logx.ErrorfContext(saveCtx, "[headless-run] incr conversation token usage error: %v", incrErr)
			}
		}
	}

	result := &headless.RunResult{
		ConversationId: convId,
		TurnId:         turnId,
		Status:         status,
		ProfileId:      profileId,
		RunKind:        req.RunKind,
		ToolErrors:     toolErrors,
	}
	if res != nil {
		result.Output = res.Output
		result.Usage = res.Usage
	}
	if runErr != nil {
		logx.WarnfContext(ctx, "[headless-run] finished with error: profile=%s, kind=%s, status=%s, err=%v",
			profileId, req.RunKind, status, runErr)
		return result, runErr
	}
	logx.InfofContext(ctx, "[headless-run] finished: profile=%s, kind=%s, status=%s, convId=%d, turnId=%s",
		profileId, req.RunKind, status, convId, turnId)
	return result, nil
}

// resolveRunStatus 由运行错误与 ctx 状态细分终态
func resolveRunStatus(ctx context.Context, runErr error) string {
	if runErr == nil {
		return headless.RunStatusSucceeded
	}
	if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return headless.RunStatusTimeout
	}
	if errors.Is(runErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return headless.RunStatusCanceled
	}
	return headless.RunStatusFailed
}
