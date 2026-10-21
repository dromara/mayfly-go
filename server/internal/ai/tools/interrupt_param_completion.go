package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mayfly-go/internal/ai/protocol"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// resumeDataCache 直接缓存 resume 数据，按 toolCallId 索引。
// 解决 eino 框架 address-based resume matching 在复杂 agent 拓扑下不可靠的问题。
// 注意：仅按 toolCallId 精确匹配（toolCallId 全局唯一），不做任何跨调用 fallback，
// 且条目带过期时间并在消费后删除，避免跨会话/历史数据污染与内存泄漏。
var resumeDataCache = struct {
	mu     sync.RWMutex
	byTool map[string]cachedResume // key: toolCallId
}{}

const resumeDataCacheTTL = time.Hour

type cachedResume struct {
	data      *ParamCompletionResume
	expiresAt time.Time
}

// contextKeyParamCompletionResume 用于通过 Go context 传递 resume 数据（最可靠的 fallback）
type contextKeyParamCompletionResume struct{}

// CacheParamCompletionResume 缓存参数补全 resume 数据（由中断扩展在 ResumeWithParams 前调用）
func CacheParamCompletionResume(toolCallId string, data *ParamCompletionResume) {
	if data == nil || toolCallId == "" {
		return
	}
	resumeDataCache.mu.Lock()
	defer resumeDataCache.mu.Unlock()
	if resumeDataCache.byTool == nil {
		resumeDataCache.byTool = make(map[string]cachedResume)
	}
	// 顺带清理过期条目，避免泄漏
	now := time.Now()
	for k, v := range resumeDataCache.byTool {
		if now.After(v.expiresAt) {
			delete(resumeDataCache.byTool, k)
		}
	}
	resumeDataCache.byTool[toolCallId] = cachedResume{data: data, expiresAt: now.Add(resumeDataCacheTTL)}
	logx.InfofContext(context.Background(), "[CacheParamCompletionResume] cached resume data for toolCallId=%s, action=%s", toolCallId, data.Action)
}

// getCachedResumeData 从缓存中获取 resume 数据（仅精确匹配 toolCallId）
func getCachedResumeData(toolCallId string) *ParamCompletionResume {
	if toolCallId == "" {
		return nil
	}
	resumeDataCache.mu.RLock()
	defer resumeDataCache.mu.RUnlock()
	c, ok := resumeDataCache.byTool[toolCallId]
	if !ok || time.Now().After(c.expiresAt) {
		return nil
	}
	return c.data
}

// WithParamCompletionResumeCtx 将 param completion resume 数据存入 Go context（最可靠的传递方式）
func WithParamCompletionResumeCtx(ctx context.Context, data *ParamCompletionResume) context.Context {
	if data == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKeyParamCompletionResume{}, data)
}

// getParamCompletionResumeFromCtx 从 Go context 中获取 param completion resume 数据
func getParamCompletionResumeFromCtx(ctx context.Context) *ParamCompletionResume {
	if v := ctx.Value(contextKeyParamCompletionResume{}); v != nil {
		if d, ok := v.(*ParamCompletionResume); ok {
			return d
		}
	}
	return nil
}

// removeCachedResumeData 移除缓存中的 resume 数据
func removeCachedResumeData(toolCallId string) {
	if toolCallId == "" {
		return
	}
	resumeDataCache.mu.Lock()
	defer resumeDataCache.mu.Unlock()
	delete(resumeDataCache.byTool, toolCallId)
}

// CompletionOption 可选项（ask_user options 模式）
type CompletionOption struct {
	Label string `json:"label"` // 显示标签
	Value string `json:"value"` // 值（JSON 字符串，前端解析为 payload）
}

// CompletionParamInfo 参数补全信息
type CompletionParamInfo struct {
	Param string `json:"param"` // 参数名
	Name  string `json:"name"`  // 参数描述
}

// ParamCompletionInterruptInfo 参数完善中断信息
type ParamCompletionInterruptInfo struct {
	BaseInterruptInfo
	// 参数类型，如"db"、"machine"、"table"等
	ParamType     string                `json:"paramType"`
	MissingParams []CompletionParamInfo `json:"missingParams"` // 缺失参数列表
	Options       []CompletionOption    `json:"options"`       // 可选项列表（通用 ask_user 模式）
}

// TryApplyResumedParams 在工具参数检查前调用，尝试从 resume 上下文或已保存的中断消息中
// 恢复之前用户补全的参数值。解决 eino 框架 resume 时使用原始 arguments 重新调用工具的问题。
// 返回 true 表示参数已成功恢复，工具应继续执行。
func TryApplyResumedParams(ctx context.Context, args any) bool {
	toolCallId := compose.GetToolCallID(ctx)
	logx.DebugfContext(ctx, "[TryApplyResumedParams] toolCallId=%s, args=%v", toolCallId, jsonx.ToStr(args))

	// 1. 先检查直接缓存（由中断扩展在 ResumeWithParams 前写入，最可靠）
	if cached := getCachedResumeData(toolCallId); cached != nil {
		if cached.Action == protocol.ActionComplete {
			if applyErr := applyResumedPayload(ctx, toolCallId, cached.Payload, args); applyErr == nil {
				logx.InfofContext(ctx, "[TryApplyResumedParams] applied from cache: %v", jsonx.ToStr(args))
				removeCachedResumeData(toolCallId)
				return true
			} else {
				logx.WarnfContext(ctx, "[TryApplyResumedParams] cache apply failed: %v", applyErr)
			}
		}
	}

	// 2. 检查已保存的中断消息（之前的 resume 已保存）
	messages, err := session.DefaultSessionStore.GetMessage(ctx, &session.MessageQuery{
		MessageType: string(InterruptTypeParamCompletion),
		ToolCallId:  toolCallId,
	})
	if err != nil {
		logx.DebugfContext(ctx, "[TryApplyResumedParams] query saved interrupt messages: %v", err)
	}
	for _, msg := range messages {
		var resumeInfo ParamCompletionResume
		if err := msg.Extra.Unmarshal(session.ExtraKeyResumeInfo, &resumeInfo); err != nil {
			continue
		}
		if resumeInfo.Action == protocol.ActionComplete {
			if err := applyResumedPayload(ctx, toolCallId, resumeInfo.Payload, args); err == nil {
				logx.InfofContext(ctx, "[TryApplyResumedParams] applied from saved message: %v", jsonx.ToStr(args))
				return true
			}
		}
	}

	// 3. 检查 Go context 传递的 resume 数据（由 agent.Run 直接设置，比 eino address matching 更可靠）
	if ctxData := getParamCompletionResumeFromCtx(ctx); ctxData != nil {
		logx.InfofContext(ctx, "[TryApplyResumedParams] found context resume data, action=%s, payload=%v", ctxData.Action, jsonx.ToStr(ctxData.Payload))
		if ctxData.Action == protocol.ActionComplete {
			if applyErr := applyResumedPayload(ctx, toolCallId, ctxData.Payload, args); applyErr == nil {
				logx.InfofContext(ctx, "[TryApplyResumedParams] applied from context: %v", jsonx.ToStr(args))
				return true
			} else {
				logx.WarnfContext(ctx, "[TryApplyResumedParams] context apply failed: %v", applyErr)
			}
		}
	}

	// 4. 检查 eino resume 上下文（最终 fallback）
	wasInterrupted, hasState, _ := tool.GetInterruptState[string](ctx)
	logx.InfofContext(ctx, "[TryApplyResumedParams] wasInterrupted=%v, hasState=%v", wasInterrupted, hasState)
	if !wasInterrupted {
		return false
	}
	isTarget, hasData, data := tool.GetResumeContext[*ParamCompletionResume](ctx)
	logx.InfofContext(ctx, "[TryApplyResumedParams] isTarget=%v, hasData=%v, data=%v", isTarget, hasData, jsonx.ToStr(data))
	if !isTarget || !hasData || data == nil {
		return false
	}
	if data.Action != protocol.ActionComplete {
		return false
	}
	if err := applyResumedPayload(ctx, toolCallId, data.Payload, args); err != nil {
		logx.WarnfContext(ctx, "[TryApplyResumedParams] failed to apply: %v", err)
		return false
	}
	logx.InfofContext(ctx, "[TryApplyResumedParams] applied from resume context: %v", jsonx.ToStr(args))
	return true
}

// applyResumedPayload 合并用户补全的参数值，并把合并后的最终参数回写进历史 tool_call 记录。
//
// 为什么「合并」与「回写」必须是同一个动作：补全值只写进内存 args 时，工具能拿到真凭证执行成功，
// 但历史里那条 tool_call 的 arguments 仍是模型最初提交的空值——审计与前端回显同实际执行不一致
// （真机核实：output 里是实际连接的凭证，payload 的 authCertName 却为空）。回写原先只挂在
// ResumeParamCompletion 的 eino 分支上，而真机实际走的是 TryApplyResumedParams 的缓存分支：它
// apply 成功后直接 return，整段回写被跳过。绑定到本函数后，任何恢复来源都不会再次漏掉回写。
func applyResumedPayload(ctx context.Context, toolCallId string, payload collx.M, args any) error {
	if err := applyPayloadToArgs(payload, args); err != nil {
		return err
	}
	writeBackCompletedArgs(ctx, toolCallId, args)
	return nil
}

// writeBackCompletedArgs 按 toolCallId 定位 tool_call 记录，把补全后的最终参数回写落库。
//
// 不回写 error：参数已在内存里补全、执行链路正常，存储层抖动不该让已成功的工具调用反而报错；
// 但必须留痕，否则历史一致性失效时日志上无从发现（本缺陷就是靠这些留痕定位的）
func writeBackCompletedArgs(ctx context.Context, toolCallId string, args any) {
	if toolCallId == "" {
		return
	}
	// 与并发写同一 Turn 的其他回写互斥，防止 RMW 覆盖。
	//
	// 这里的「读整行-改-写整行」与 EventMapper 的终态写入（UpdateResumedToolCallItem，不加此锁）
	// 不构成竞态，依据是不变量：**同一条 tool_call 记录上，本回写恒早于其终态写入** —— 回写在工具
	// 体内同步执行（TryApplyResumedParams 的第一步），终态事件只可能在该工具返回之后产生；并行多工具
	// 各自命中不同的 item 行。真机日志实证该顺序（17:22:02.325 回写 UPDATE → 17:22:02.435 终态读到已
	// 回写的值）。若将来改成「参数补全异步回写」或「终态在工具返回前抢占落库」，必须让两侧共用此锁
	session.WithTurnLock(ctx, func() {
		toolCallMsgs, err := session.DefaultSessionStore.GetMessage(ctx, &session.MessageQuery{
			MessageType: session.MsgTypeToolCall,
			ToolCallId:  toolCallId,
		})
		if err != nil || len(toolCallMsgs) == 0 {
			logx.WarnfContext(ctx, "[paramCompletion] no tool_call message to write back, toolCallId=%s, err=%v", toolCallId, err)
			return
		}
		written := false
		for _, toolCallMsg := range toolCallMsgs {
			for i := range toolCallMsg.ToolCalls {
				if toolCallMsg.ToolCalls[i].ID != toolCallId {
					continue
				}
				toolCallMsg.ToolCalls[i].Function.Arguments = jsonx.ToStr(args)
				if err := session.DefaultSessionStore.UpdateMessage(ctx, toolCallMsg); err != nil {
					logx.ErrorfContext(ctx, "update tool call arguments failed: %v", err)
					continue
				}
				written = true
				logx.DebugfContext(ctx, "[paramCompletion] wrote back args, msgId=%d toolCallId=%s args=%s",
					toolCallMsg.Id, toolCallId, toolCallMsg.ToolCalls[i].Function.Arguments)
			}
		}
		if !written {
			// 留痕：toolCallId 对不上（中断记录与助手调用的 id 不同源时）同样是静默失效
			logx.WarnfContext(ctx, "[paramCompletion] toolCallId %q not matched in %d tool_call messages, "+
				"arguments kept pre-completion values", toolCallId, len(toolCallMsgs))
		}
	})
}

// applyPayloadToArgs 将 payload 中的参数值合并到 args（覆盖同名字段，保留未提及字段，
// 避免 payload 只包含部分参数时将 args 原有值清零）
func applyPayloadToArgs(payload collx.M, args any) error {
	paramValuesMap, err := extractParamValues(payload)
	if err != nil {
		return err
	}
	return mergeParamsIntoArgs(paramValuesMap, args)
}

// extractParamValues 从 payload 中提取参数值 map（payload 为扁平结构：
// 前端提交的 { ...inputValues, caches }）
func extractParamValues(payload collx.M) (map[string]any, error) {
	if payload == nil {
		return nil, fmt.Errorf("nil payload")
	}
	// 排除元数据字段，其余键值即为参数
	paramValuesMap := make(map[string]any, len(payload))
	for k, v := range payload {
		if k == "id" || k == "displayName" || k == "caches" {
			continue
		}
		paramValuesMap[k] = v
	}
	if len(paramValuesMap) == 0 {
		return nil, fmt.Errorf("no valid params in payload")
	}
	return paramValuesMap, nil
}

// mergeParamsIntoArgs 将参数 map 合并到 args：先序列化 args 为 map，再覆盖 payload 提供的字段，
// 最后反序列化回 args，确保未提及的字段保留原值
func mergeParamsIntoArgs(paramValuesMap map[string]any, args any) error {
	argsMap := make(map[string]any)
	if raw := jsonx.ToStr(args); raw != "" && raw != "null" {
		if err := json.Unmarshal([]byte(raw), &argsMap); err != nil {
			return fmt.Errorf("unmarshal args to map: %w", err)
		}
	}
	for k, v := range paramValuesMap {
		argsMap[k] = v
	}
	return json.Unmarshal([]byte(jsonx.ToStr(argsMap)), args)
}

// InterruptOrResumeParamCompletion 中断或恢复参数完善
func InterruptOrResumeParamCompletion(ctx context.Context, toolDesc string, args any, reason string, paramType string, missingParams []CompletionParamInfo, options ...[]CompletionOption) error {
	// 取消决策必须在「重新发起中断」之前消费掉。
	//
	// 真机实测：连续点 4 次「取消」，后端每次都确实收到了 action=cancel 的 resume（日志有
	// CacheParamCompletionResume 按 toolCallId 缓存），但 TryApplyResumedParams 只认 complete、
	// handleParamCompletion 又依赖 eino 的 target 匹配（日志里存在 isTarget=false 的场次），
	// 于是工具回头重走参数检查、再弹一次补全：中断永不解除、取消决策永不开花，
	// 且前端全程静默无任何报错
	if cancelled := resumedCancellationDecision(ctx); cancelled != nil {
		return NewToolError(cancelled, RecoverNone)
	}
	// 注意：TryApplyResumedParams 应在工具参数检查前由工具自身调用，
	// 到达此处说明参数仍不完整，需要中断
	isResume, err := ResumeParamCompletion(ctx, args)
	if !isResume {
		var opts []CompletionOption
		if len(options) > 0 {
			opts = options[0]
		}
		// 无人值守运行里弹卡片 = 把整轮挂在一个永不应答的中断上（真机：事件触发的诊断运行
		// 卡在 authCertName 为空，整轮 interrupted、运行记 failed、摘要就是那句「请完善参数」，
		// 7139 tokens 花完零产出）。这里改为把「缺哪些参数 + 有哪些可选值」直接回传给模型：
		// 模型上一步已经调过 ListResources，完全有能力自己补全；RecoverRetry 与工具体内
		// 「缺必填参数要求重试」同一恢复语义，不会把运行整体判为不可重试
		if IsUnattended(ctx) {
			// 留痕：无人值守下模型提交缺参调用本身就是要关注的事件（此前它表现为「运行白烧完 token
			// 却零产出」，而日志里什么也不会留下，只能去翻 turn_item 才能定性）
			logx.WarnfContext(ctx, "[paramCompletion] unattended run: required params missing, returned retryable tool error instead of interrupt: tool=%s, paramType=%s, missing=%v",
				toolDesc, paramType, missingParams)
			return NewToolError(unattendedCompletionMsg(toolDesc, missingParams, opts), RecoverRetry)
		}
		return InterruptParamCompletion(ctx, toolDesc, args, reason, paramType, missingParams, opts)
	}
	if err == nil {
		return nil
	}
	return NewToolError(err, RecoverNone)
}

// unattendedCompletionMsg 无人值守下缺参的提示文本。
//
// 面向模型而不是用户，所以不进 i18n：写进字典就会随服务端默认语言漂移，而这里要的是
// 「模型下一轮能据以行动」的稳定指令语义（与 sql_exec 的 remark 必填提示同一形态）。
// 可选值只列 Label：Value 是给前端解析的 payload JSON，拿去让模型回填只会多一层无意义格式
func unattendedCompletionMsg(toolDesc string, missingParams []CompletionParamInfo, opts []CompletionOption) error {
	missing := make([]string, 0, len(missingParams))
	for _, param := range missingParams {
		missing = append(missing, param.Param+("（"+param.Name+"）"))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "工具 %s 缺少必填参数：%s。本次为无人值守运行，没有可以回答参数补全的人，不会弹出补全卡片。", toolDesc, strings.Join(missing, "、"))
	if len(opts) > 0 {
		labels := make([]string, 0, len(opts))
		for _, opt := range opts {
			labels = append(labels, opt.Label)
		}
		fmt.Fprintf(&sb, "候选值：%s。请用上述候选之一重新提交本工具调用；若无法确定目标，改用只读工具完成任务，或将该动作登记为变更提案等待人工审批，不要空转重试", strings.Join(labels, " | "))
	}
	return fmt.Errorf("%s", sb.String())
}

// InterruptParamCompletion 中断参数完善
func InterruptParamCompletion(ctx context.Context, toolDesc string, args any, reason string, paramType string, missingParams []CompletionParamInfo, opts []CompletionOption) error {
	argsInJSON := jsonx.ToStr(args)
	// 创建中断信息（包含完整的MissingParams和Options）
	interruptInfo := &ParamCompletionInterruptInfo{
		BaseInterruptInfo: BaseInterruptInfo{
			Type:        InterruptTypeParamCompletion,
			Title:       i18n.T(imsg.InfoIncomplete),
			Description: reason,
			Payload:     missingParams,
			ToolCallId:  compose.GetToolCallID(ctx),
			ToolInfo:    &ToolInfo{Name: toolDesc},
			Arguments:   argsInJSON,
		},
		ParamType:     paramType,
		MissingParams: missingParams,
		Options:       opts,
	}

	return tool.StatefulInterrupt(ctx, interruptInfo, argsInJSON)
}

// ResumeParamCompletion 恢复参数完善
func ResumeParamCompletion(ctx context.Context, args any) (bool, error) {
	// 首先检查是否有参数补全过的中断消息，并从中提取参数值进行恢复
	messages, _ := session.DefaultSessionStore.GetMessage(ctx, &session.MessageQuery{MessageType: string(InterruptTypeParamCompletion), ToolCallId: compose.GetToolCallID(ctx)})
	if len(messages) > 0 {
		for _, msg := range messages {
			var resumeInfo ParamCompletionResume
			if err := msg.Extra.Unmarshal(session.ExtraKeyResumeInfo, &resumeInfo); err != nil {
				// unmarshal 失败说明不是有效的 resume 消息，继续检查下一条
				continue
			}
			return true, handleParamCompletion(ctx, &resumeInfo, args)
		}
		// 所有消息都检查完毕但没有有效的 resume info，视为无 resume
		return false, nil
	}

	// 检查是否是从中断恢复
	wasInterrupted, _, _ := tool.GetInterruptState[string](ctx)
	if !wasInterrupted {
		return false, nil
	}

	// 直接使用 GetResumeContext 检查参数补全恢复
	isTarget, hasData, data := tool.GetResumeContext[*ParamCompletionResume](ctx)
	if !isTarget || !hasData {
		// 不是参数补全目标，继续执行
		return false, nil
	}

	// 把用户决策写回中断消息，供历史回显与重试读取；arguments 的回写已由 handleParamCompletion
	// 里的 applyResumedPayload 统一完成（所有恢复来源走同一段回写，不再分支各持）
	if AppendResumeInfo(ctx, data.InterruptId, data) == nil {
		// 留痕而非静默：找不到中断消息时这张卡片在历史里停在「待补全」，用户决策不落地
		logx.WarnfContext(ctx, "[paramCompletion] interrupt message not found, interruptId=%s, turnId=%s",
			data.InterruptId, data.TurnId)
	}

	// 无论是否找到历史消息，都要更新参数（含把补全后的最终参数回写进 tool_call 记录）
	if err := handleParamCompletion(ctx, data, args); err != nil {
		return true, err
	}

	return true, nil
}

func handleParamCompletion(ctx context.Context, data *ParamCompletionResume, args any) error {
	if data.Action != protocol.ActionComplete {
		return paramCompletionCancelled()
	}
	logx.InfofContext(ctx, "[ParamCompletion] payload: %v", jsonx.ToStr(data.Payload))

	if err := applyResumedPayload(ctx, compose.GetToolCallID(ctx), data.Payload, args); err != nil {
		return err
	}

	logx.InfofContext(ctx, "[ParamCompletion] args after merge: %v", jsonx.ToStr(args))
	return nil
}

// paramCompletionCancelledText 取消参数补全时回给模型的固定文案（两处消费同源，不各自拼字符串）
const paramCompletionCancelledText = CancelledToolPrefix + " The user has cancelled the parameter completion for this tool.\n" +
	"Please do not retry parameter completion automatically. Ask the user for further instructions if needed."

func paramCompletionCancelled() error {
	return errors.New(paramCompletionCancelledText)
}

// resumedCancellationDecision 若本次工具调用已收到「取消」决策，返回对应的拒绝错误（并清除缓存，
// 避免下一次工具调用误读旧决策）。
//
// 只认明确的 action=cancel：未知 action 不在此处拦（交给 handleParamCompletion 的兼容分支），
// 否则一个脏值就能把正常补全流程打死。取数走缓存与 ctx 两个通道（与 TryApplyResumedParams
// 同源），不依赖 eino 的 address matching——实测那个 matching 会给出 isTarget=false
func resumedCancellationDecision(ctx context.Context) error {
	toolCallId := compose.GetToolCallID(ctx)

	// 缓存通道按 toolCallId 精确匹配：取消是一次性的，消费后必须摘除，否则会外溢给
	// 同一轮里后续另一个需要补全的工具调用（那等于用户从未做过却被打上「用户已取消」）
	if cached := getCachedResumeData(toolCallId); cached != nil {
		if cached.Action == protocol.ActionComplete {
			// 补全交由 TryApplyResumedParams 消费（含 apply 失败后保留缓存重试），这里不插手
			return nil
		}
		removeCachedResumeData(toolCallId)
		return paramCompletionCancelled()
	}

	// ctx 通道挂的是**整轮**上下文、无法按 toolCallId 摘除，因此只在拿不到 toolCallId 时兜底；
	// 判据与 handleParamCompletion 一致（!= complete 即视为非完成），避免前端新增 action 值时
	// 这里放行、那边拒绝，落回「重新发起中断」的死循环
	if toolCallId == "" {
		if ctxData := getParamCompletionResumeFromCtx(ctx); ctxData != nil && ctxData.Action != protocol.ActionComplete {
			return paramCompletionCancelled()
		}
	}
	return nil
}

// paramCompletionExtension 参数补全中断的扩展实现（经中断扩展贡献者统一装配，见 agent/ext/interrupt）
type paramCompletionExtension struct{}

var _ InterruptExtension = (*paramCompletionExtension)(nil)

func (e *paramCompletionExtension) ConvertResume(resume *InterruptResume) any {
	return &ParamCompletionResume{InterruptResume: resume}
}

func (e *paramCompletionExtension) PrepareResumeCtx(ctx context.Context, resume *InterruptResume) context.Context {
	target := e.ConvertResume(resume)
	pcResume, ok := target.(*ParamCompletionResume)
	if !ok {
		return ctx
	}
	// 方式1: 通过 Go context 传递（最可靠）
	ctx = WithParamCompletionResumeCtx(ctx, pcResume)
	// 方式2: 按 toolCallId 缓存，供工具执行时 TryApplyResumedParams 使用
	if msgs, err := session.DefaultSessionStore.GetMessage(ctx, &session.MessageQuery{ActionId: resume.InterruptId}); err == nil && len(msgs) > 0 {
		if tcId := msgs[0].ToolCallId; tcId != "" {
			CacheParamCompletionResume(tcId, pcResume)
			logx.InfofContext(ctx, "[paramCompletionExtension] cached param completion resume for toolCallId=%s (interruptId=%s)", tcId, resume.InterruptId)
		}
	}
	return ctx
}

func (e *paramCompletionExtension) ExtraEventMetadata(info InterruptMetadata) collx.M {
	pci, ok := info.(*ParamCompletionInterruptInfo)
	if !ok {
		return nil
	}
	meta := collx.M{"paramType": pci.ParamType}
	if len(pci.Options) > 0 {
		meta["options"] = pci.Options
	}
	return meta
}

// NewParamCompletionExtension 创建参数补全中断扩展（经中断扩展贡献者在装配期装载）
func NewParamCompletionExtension() InterruptExtension { return &paramCompletionExtension{} }
