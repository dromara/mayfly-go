package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// CompletionOption 可选项（对齐 tokhub ask_user options 模式）
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
		if cached.Action == "complete" {
			if applyErr := applyPayloadToArgs(cached.Payload, args); applyErr == nil {
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
		if err := msg.Extra.Unmarshal("resumeInfo", &resumeInfo); err != nil {
			continue
		}
		if resumeInfo.Action == "complete" {
			if err := applyPayloadToArgs(resumeInfo.Payload, args); err == nil {
				logx.InfofContext(ctx, "[TryApplyResumedParams] applied from saved message: %v", jsonx.ToStr(args))
				return true
			}
		}
	}

	// 3. 检查 Go context 传递的 resume 数据（由 agent.Run 直接设置，比 eino address matching 更可靠）
	if ctxData := getParamCompletionResumeFromCtx(ctx); ctxData != nil {
		logx.InfofContext(ctx, "[TryApplyResumedParams] found context resume data, action=%s, payload=%v", ctxData.Action, jsonx.ToStr(ctxData.Payload))
		if ctxData.Action == "complete" {
			if applyErr := applyPayloadToArgs(ctxData.Payload, args); applyErr == nil {
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
	if data.Action != "complete" {
		return false
	}
	if err := applyPayloadToArgs(data.Payload, args); err != nil {
		logx.WarnfContext(ctx, "[TryApplyResumedParams] failed to apply: %v", err)
		return false
	}
	logx.InfofContext(ctx, "[TryApplyResumedParams] applied from resume context: %v", jsonx.ToStr(args))
	return true
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
	// 注意：TryApplyResumedParams 应在工具参数检查前由工具自身调用，
	// 到达此处说明参数仍不完整，需要中断
	isResume, err := ResumeParamCompletion(ctx, args)
	if !isResume {
		var opts []CompletionOption
		if len(options) > 0 {
			opts = options[0]
		}
		return InterruptParamCompletion(ctx, toolDesc, args, reason, paramType, missingParams, opts)
	}
	if err == nil {
		return nil
	}
	return NewToolError(err, RecoverNone)
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
			if err := msg.Extra.Unmarshal("resumeInfo", &resumeInfo); err != nil {
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

	// 修改参数调用的消息体，更新参数后
	msg := AppendResumeInfo(ctx, data.InterruptId, data)

	// 无论是否找到历史消息，都要更新参数
	if err := handleParamCompletion(ctx, data, args); err != nil {
		return true, err
	}

	// 如果找到了历史消息，更新 tool_call 的 arguments
	if msg != nil {
		// 对同一 TurnId 的 RMW 操作加锁，防止并发覆盖
		session.WithTurnLock(ctx, func() {
			toolCallMsgs, err := session.DefaultSessionStore.GetMessage(ctx, &session.MessageQuery{TurnId: data.TurnId, MessageType: "tool_call"})
			if err != nil || len(toolCallMsgs) == 0 {
				return
			}

			for _, toolCallMsg := range toolCallMsgs {
				for i := range toolCallMsg.ToolCalls {
					if toolCallMsg.ToolCalls[i].ID == msg.ToolCallId {
						toolCallMsg.ToolCalls[i].Function.Arguments = jsonx.ToStr(args)
						if err := session.DefaultSessionStore.UpdateMessage(ctx, toolCallMsg); err != nil {
							logx.ErrorfContext(ctx, "update tool call arguments failed: %v", err)
						}
						break
					}
				}
			}
		})
	}

	return true, nil
}

func handleParamCompletion(ctx context.Context, data *ParamCompletionResume, args any) error {
	if data.Action != "complete" {
		return errors.New("[PARAM_COMPLETION_CANCELLED] The user has cancelled the parameter completion for this tool.\nPlease do not retry parameter completion automatically. Ask the user for further instructions if needed.")
	}

	logx.InfofContext(ctx, "[ParamCompletion] payload: %v", jsonx.ToStr(data.Payload))

	if err := applyPayloadToArgs(data.Payload, args); err != nil {
		return err
	}

	logx.InfofContext(ctx, "[ParamCompletion] args after merge: %v", jsonx.ToStr(args))
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
