package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/api/form"
	"mayfly-go/internal/ai/application/dto"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/validatorx"
	"mayfly-go/pkg/ws"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/gorilla/websocket"
)

// chatRt 会话级运行中 turn 注册表（api 包级单例）
var chatRt = newChatRuntime()

// Chat WebSocket 聊天，使用 EventMsg 结构化事件协议。
//
// turn 运行与 WS 连接解耦（对齐 tokhub turn registry + replay）：
//   - agent Run 在独立 goroutine 执行，事件经 turnEventBus 广播（全量缓冲），
//     连接断开/页面刷新不影响 turn 运行
//   - stop 消息显式取消 turn ctx（传播到 LLM 请求与工具执行），是唯一真正中断途径
//   - attach 消息订阅运行中 turn：先回放缓冲快照，再持续订阅实时事件续上流式输出
func (a *Ai) Chat(rc *req.Ctx) {
	wsConn, err := ws.Upgrader.Upgrade(rc.GetWriter(), rc.GetRequest(), nil)
	if err != nil {
		biz.ErrIsNilAppendErr(err, "Upgrade websocket fail: %s")
	}
	if err := req.PermissionHandler(rc); err != nil {
		biz.ErrIsNil(err)
	}
	w := &wsWriter{conn: wsConn}
	defer func() {
		if rec := recover(); rec != nil {
			w.Write(protocol.NewErrorEvent(anyx.ToString(rec), "server"))
		}
		wsConn.Close()
	}()

	ag, err := agent.GetDefaultAgent(rc.MetaCtx)
	biz.ErrIsNilAppendErr(err, "get agent error: %s")

	// turn 的 baseCtx 与 WS 连接生命周期解耦：连接断开（刷新页面）不取消运行中 turn，
	// 仅显式 stop 取消；登录身份/链路 ID 显式透传（hijacked 连接的请求 ctx 不可依赖）
	baseCtx := contextx.WithTraceId(context.Background())
	if la := contextx.GetLoginAccount(rc.MetaCtx); la != nil {
		baseCtx = contextx.WithLoginAccount(baseCtx, la)
	}

	// 连接当前订阅的 turn（切换订阅前取消旧订阅）
	var curSub *turnSubscription
	cancelSub := func() {
		if curSub != nil {
			curSub.unsub()
			curSub = nil
		}
	}
	defer cancelSub()

	// 心跳保活：本连接不走 pkg/ws ClientManager 心跳体系，代理/服务端空闲超时会
	// 杀掉静默连接（前端半开无感知，直到下次 attach/send 探测才暴露为「连接已断开」）。
	// 定期下发 heartbeat 保持活跃；写失败（半开）主动断连，触发前端重连
	stopHeartbeat := make(chan struct{})
	defer close(stopHeartbeat)
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if w.Write(&protocol.EventMsg{Type: protocol.EventTypeHeartbeat}) != nil {
					w.Close()
					return
				}
			case <-stopHeartbeat:
				return
			}
		}
	}()

	for {
		messageType, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Debugf("ws close: %v", err)
			}
			break
		}
		if messageType != websocket.TextMessage {
			continue
		}

		// WS 循环内禁止 panic 中断连接，解析/校验失败转为错误事件并继续读取下一条消息
		chatReq, err := jsonx.To[form.ChatRequest](message)
		if err != nil {
			w.Write(protocol.NewErrorEvent(fmt.Sprintf("parse chat request error: %s", err), "server"))
			continue
		}
		if err := validatorx.Validate(chatReq); err != nil {
			w.Write(protocol.NewErrorEvent(fmt.Sprintf("invalid chat request: %s", err), "server"))
			continue
		}

		switch chatReq.Type {
		case form.ChatMsgTypeStop:
			// 显式 stop：取消 turn ctx（Run 阻塞在 goroutine，不影响读取下一条消息）
			if !chatRt.Stop(chatReq.ConversationId) {
				w.Write(protocol.NewErrorEvent("no running turn to stop", "server"))
			}
			continue
		case form.ChatMsgTypeAttach:
			// 订阅运行中 turn：先回放缓冲快照，再持续订阅实时事件（刷新/重连续流）
			cancelSub()
			curSub = a.attachRunningTurn(w, chatReq.ConversationId)
			continue
		case form.ChatMsgTypeText, form.ChatMsgTypeInterruptResume:
			if strings.TrimSpace(chatReq.Content) == "" {
				w.Write(protocol.NewErrorEvent("content is required", "server"))
				continue
			}
		default:
			w.Write(protocol.NewErrorEvent(fmt.Sprintf("unknown chat type: %s", chatReq.Type), "server"))
			continue
		}

		// 解析或创建 conversation
		convId := chatReq.ConversationId
		if convId <= 0 {
			conv, createErr := a.conversationApp.CreateConversation(rc.MetaCtx, &dto.CreateConversationReq{})
			if createErr != nil {
				w.Write(protocol.NewErrorEvent(fmt.Sprintf("create conversation error: %s", createErr), "server"))
				continue
			}
			convId = conv.Id
			if w.Write(protocol.NewConversationCreatedEvent(convId)) != nil {
				break
			}
		}

		turnId := stringx.SortableUUID()

		// 中断恢复：先解析恢复参数，turnId 统一使用恢复目标的 turn，
		// 保证前端 activeTurnId 与后续所有事件的 turnId 一致
		var resumeParams []any
		if chatReq.Type == form.ChatMsgTypeInterruptResume {
			resumeList, resumeErr := parseResumeList(chatReq.Content)
			if resumeErr != nil {
				w.Write(protocol.NewErrorEvent(resumeErr.Error(), "server"))
				continue
			}
			resumeParams = make([]any, len(*resumeList))
			for i := range *resumeList {
				resumeParams[i] = &(*resumeList)[i]
			}
			turnId = (*resumeList)[0].TurnId
		}

		// 会话级互斥注册（Start 时创建 turn ctx 与事件总线；userId 用于运行中列表按用户过滤）
		userId := ""
		if la := contextx.GetLoginAccount(rc.MetaCtx); la != nil {
			userId = fmt.Sprintf("%d", la.Id)
		}
		turn, startErr := chatRt.Start(convId, turnId, userId, baseCtx)
		if startErr != nil {
			w.Write(protocol.NewErrorEvent(startErr.Error(), "server"))
			continue
		}

		var userMessage []adk.Message
		if resumeParams == nil {
			// 资源/技能芯片引用渲染为文本标记随消息下发，使模型明确目标资源，
			// 避免工具调用因资源参数缺失而中断等待用户手动补充；
			// 渲染经 protocol 注册式分发（新增芯片/资源类型零修改本层）。
			// 保存用户消息为 TurnItem：content 用结构化 segments（芯片引用保留元数据，
			// 历史回显可恢复芯片样式，对齐 tokhub ContentSegment 贯穿设计）；
			// 发给 LLM 的完整定位标识由 RenderSegments 注入，两者职责分离
			userSegments := buildUserSegments(chatReq.Content, chatReq.Segments)
			userMessage = collx.AsArray(schema.UserMessage(protocol.RenderSegments(userSegments)))
			userItem := protocol.NewMessageTurnItem(stringx.SortableUUID(), string(schema.User), userSegments)
			turn.CollectItems(toEntityTurnItem(convId, turnId, userItem, entity.ItemStatusSuccess))
		}

		// turn_started 进入事件总线（发起连接经回放接收，其它订阅者实时接收）
		turn.Publish(protocol.NewTurnStartedEvent(turnId, convId))

		// 独立 goroutine 执行：连接断开/页面刷新不影响 turn 运行
		go a.runTurn(turn, ag, chatReq, convId, turnId, resumeParams, userMessage)

		// 本连接订阅该 turn（回放含 turn_started，实时事件无缝续上）
		cancelSub()
		curSub = turn.Subscribe(protocol.NewTurnAttachedEvent(turnId, convId))
		go pumpTurnSubscription(w, curSub)
	}
}

// attachRunningTurn 处理 attach：会话存在运行中 turn 时订阅（回放缓存快照后续流），否则回复 turn_not_running
func (a *Ai) attachRunningTurn(w *wsWriter, convId uint64) *turnSubscription {
	rt := chatRt.GetRunning(convId)
	if rt == nil {
		w.Write(protocol.NewTurnNotRunningEvent(convId))
		return nil
	}
	sub := rt.Subscribe(protocol.NewTurnAttachedEvent(rt.turnId, convId))
	go pumpTurnSubscription(w, sub)
	return sub
}

// runTurn 执行一次 agent turn（独立 goroutine）：事件全部发布到 turn 事件总线，不直接触碰 WS 连接
func (a *Ai) runTurn(turn *runningTurn, ag *agent.Agent, chatReq *form.ChatRequest, convId uint64, turnId string, resumeParams []any, userMessage []adk.Message) {
	turnCtx := turn.ctx
	publish := turn.Publish

	eventMapper := agent.NewEventMapper()

	// 恢复路径预加载（对齐 tokhub）：将该 turn 挂起的 interrupted tool_call item
	// 注册到 EventMapper（工具完成时复用原 item_id），并建立 itemId → 恢复决策映射
	resumeByItemId := map[string]*protocol.InterruptResume{}
	if len(resumeParams) > 0 {
		a.trackResumedToolCalls(turnCtx, convId, turnId, resumeParams, eventMapper, resumeByItemId)
	}

	// 收集本轮所有 TurnItem 用于持久化（回调均由 Agent.Run 同步调用，无并发）
	turnItems := make([]*entity.TurnItem, 0)
	collect := func(items ...*entity.TurnItem) {
		turnItems = append(turnItems, items...)
	}

	agentRunOptions := []agent.RunOption{
		agent.WithRunSessionKey(fmt.Sprintf("conv:%d", convId)),
		agent.WithTurnId(turnId),
		agent.WithOnChunk(func(ctx context.Context, m adk.Message) error {
			if len(m.ToolCalls) > 0 || m.Role == schema.Tool {
				return nil
			}
			// 发布不会失败：WS 断开不影响 agent 流式输出（刷新页面后 attach 回放续上）
			for _, evt := range eventMapper.MapChunk(turnId, m) {
				publish(evt)
			}
			return nil
		}),
		agent.WithOnEvent(func(ctx context.Context, ae *adk.AgentEvent, m adk.Message) error {
			currentTurnId := agent.GetTurnId(m)
			if currentTurnId == "" {
				currentTurnId = turnId
			}
			if len(m.ToolCalls) > 0 {
				for _, evt := range eventMapper.MapToolCallEvent(currentTurnId, m) {
					publish(evt)
					// 收集 TurnItem（MapToolCallEvent 会先发 reasoning/message 的 item_completed，
					// 终态事件落 success，进行中的 tool_call 落 active）
					if evt.Item != nil {
						status := entity.ItemStatusActive
						if evt.Type == protocol.EventTypeItemCompleted {
							status = entity.ItemStatusSuccess
						}
						collect(toEntityTurnItem(convId, evt.TurnId, evt.Item, status))
					}
				}
				return nil
			}
			if m.Role == schema.Tool {
				for _, evt := range eventMapper.MapToolResultEvent(ctx, currentTurnId, m) {
					publish(evt)
					// 收集 TurnItem（使用事件中的实际状态，而非硬编码 success）
					if evt.Item != nil {
						collect(toEntityTurnItem(convId, evt.TurnId, evt.Item, toolCallItemStatus(evt.Item)))
					}
				}
				return nil
			}
			if m.Role == session.RoleInternal {
				extra := collx.M(m.Extra)
				if extra != nil && tools.IsInterruptContent(extra["content"]) {
					// 通过中断元数据泛化判断（IsInterruptContent），新增中断类型无需修改此处
					interruptEvents := eventMapper.MapInterruptEvent(currentTurnId, m)
					for _, evt := range interruptEvents {
						publish(evt)
					}
					// 对齐 tokhub：中断信息统一存到被中断工具调用的 tool_call item
					// extra 列（{"interrupt": InterruptInfo}），不产生独立 internal item
					if len(interruptEvents) > 0 && interruptEvents[0].Interrupt != nil {
						if !markInterruptedToolCall(turnItems, interruptEvents[0].Interrupt, convId) {
							logx.Warnf("interrupted tool_call item not found, toolCallId=%s",
								interruptEvents[0].Interrupt.ToolCallId)
						}
					}
				}
				return nil
			}
			return nil
		}),
	}

	if resumeParams != nil {
		agentRunOptions = append(agentRunOptions, agent.WithResumeParams(resumeParams...))
	}

	runRes, err := ag.Run(turnCtx, userMessage, agentRunOptions...)

	// 完成流式输出，收集最终的 TurnItem
	finalEvents := eventMapper.CompleteStreaming(turnId, nil)
	for _, evt := range finalEvents {
		if evt.Item != nil {
			collect(toEntityTurnItem(convId, evt.TurnId, evt.Item, entity.ItemStatusSuccess))
		}
		publish(evt)
	}

	// 收尾持久化使用无取消 ctx：turn ctx 可能已被显式 stop 取消，仍需保证落库
	saveCtx := context.WithoutCancel(turnCtx)

	// 去重：同一 ItemId 可能有多条记录（streaming 中间态 + 完成态），保留最后一条
	turnItems = append(turnItems, turn.SnapshotItems()...)
	if len(turnItems) > 0 {
		deduped := deduplicateTurnItems(turnItems)
		// 恢复路径分流：复用原 item_id 的工具调用走按行更新（BatchInsert 会产生重复行），
		// 其余新 item 批量插入
		inserts := make([]*entity.TurnItem, 0, len(deduped))
		for _, item := range deduped {
			if resume, ok := resumeByItemId[item.ItemId]; ok {
				if updErr := a.turnItemApp.UpdateResumedToolCallItem(saveCtx, convId, item, resume); updErr != nil {
					logx.Errorf("update resumed tool_call item error: %v", updErr)
				}
				continue
			}
			inserts = append(inserts, item)
		}
		if len(inserts) > 0 {
			if saveErr := a.turnItemApp.BatchSaveTurnItems(saveCtx, inserts); saveErr != nil {
				logx.Errorf("save turn items error: %v", saveErr)
			}
		}
	}

	// turn 级 token 用量：下发 TurnCompleted 事件 + 累加到会话统计字段
	var turnUsage *protocol.TurnUsage
	if runRes != nil && runRes.Usage != nil {
		turnUsage = &protocol.TurnUsage{
			InputTokens:  int64(runRes.Usage.PromptTokens),
			OutputTokens: int64(runRes.Usage.CompletionTokens),
			TotalTokens:  int64(runRes.Usage.TotalTokens),
		}
		if incrErr := a.conversationApp.IncrTokenUsage(saveCtx, convId,
			turnUsage.InputTokens, turnUsage.OutputTokens, turnUsage.TotalTokens); incrErr != nil {
			logx.Errorf("incr conversation token usage error: %v", incrErr)
		}
	}

	status := protocol.TurnStatusSuccess
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// 显式 stop 触发的取消：turn 以 stopped 终态收尾，不下发错误事件
			status = protocol.TurnStatusStopped
		} else {
			status = protocol.TurnStatusFailed
			publish(protocol.NewErrorEvent(err.Error(), "agent"))
		}
	}

	publish(protocol.NewTurnCompletedEvent(turnId, status, turnUsage))

	// turn 收尾：终结事件总线（触发订阅泵 drain 剩余事件并写 end）并移出注册表
	turn.bus.Finish()
	chatRt.Remove(convId, turnId)
}

// markInterruptedToolCall 将中断信息落到被中断工具调用的 tool_call item 上
// （对齐 tokhub：extra["interrupt"] 存 InterruptInfo，item 状态置 interrupted）。
// 中断必然由工具审批/参数补全触发，本轮已收集的 tool_call item 中必能命中；
// 返回是否命中（未命中仅告警，中断信息随事件流下发、不落库）
func markInterruptedToolCall(items []*entity.TurnItem, evt *protocol.InterruptEvent, convId uint64) bool {
	if evt == nil || evt.ToolCallId == "" {
		return false
	}
	// 倒序取最后收集的同 toolCallId item（与落库去重“保留最后一条”口径一致）
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.ItemType == entity.ItemTypeToolCall && item.ToolCallId == evt.ToolCallId {
			item.Status = protocol.TurnItemStatusInterrupted
			if item.Extra == nil {
				item.Extra = collx.M{}
			}
			item.Extra["interrupt"] = protocol.NewInterruptInfo(evt, convId)
			return true
		}
	}
	return false
}

// trackResumedToolCalls 恢复路径预加载：从持久化层加载该 turn 挂起的 interrupted
// tool_call item，注册 toolCallId → 原 itemId 到 EventMapper（工具完成时复用原
// item_id、更新同一行，对齐 tokhub execute.rs 的第一阶段挂起行加载）；同时按
// request_id 匹配恢复决策，建立 itemId → 决策映射供落库时 merge 到 interrupt.resume
func (a *Ai) trackResumedToolCalls(ctx context.Context, convId uint64, turnId string, resumeParams []any, eventMapper *agent.EventMapper, resumeByItemId map[string]*protocol.InterruptResume) {
	items, err := a.turnItemApp.SelectByTurnId(ctx, convId, turnId)
	if err != nil {
		logx.Errorf("load interrupted items for resume error: %v", err)
		return
	}

	// 恢复决策按 interruptId（即 extra.interrupt.request_id）索引
	resumeByRequestId := make(map[string]*tools.InterruptResume)
	for _, p := range resumeParams {
		if r, ok := p.(*tools.InterruptResume); ok {
			resumeByRequestId[r.InterruptId] = r
		}
	}

	for _, item := range items {
		if item.ItemType != entity.ItemTypeToolCall || item.Status != protocol.TurnItemStatusInterrupted {
			continue
		}
		info, err := jsonx.ToByStr[protocol.InterruptInfo](jsonx.ToStr(item.Extra["interrupt"]))
		if err != nil || info == nil || info.Kind == "" {
			continue
		}
		eventMapper.TrackResumedToolCall(item.ToolCallId, item.ItemId)
		if r, ok := resumeByRequestId[info.RequestId]; ok {
			resumeByItemId[item.ItemId] = protocol.NewResumeFromResumeInfo(r)
		}
	}
}

// pumpTurnSubscription 订阅泵：将 turn 事件转发到 WS 连接
//
// 顺序：回放快照 → attached 标志 → 实时事件；turn 结束后 drain 残余事件并写 end
// （连接级流结束信号，对齐此前每轮请求结束写 end 的前端语义）
func pumpTurnSubscription(w *wsWriter, sub *turnSubscription) {
	if sub == nil {
		return
	}
	defer sub.unsub()

	for _, evt := range sub.replay {
		if w.Write(evt) != nil {
			w.Close()
			return
		}
	}
	if w.Write(sub.attached) != nil {
		w.Close()
		return
	}

	for {
		select {
		case evt, ok := <-sub.ch:
			if !ok {
				// 慢消费者缓冲溢出被移除：主动断连，前端重连后 attach 回放恢复
				w.Close()
				return
			}
			if w.Write(evt) != nil {
				w.Close()
				return
			}
		case <-sub.done:
			// turn 收尾完成：drain channel 残余事件后写 end
			for {
				select {
				case evt, ok := <-sub.ch:
					if !ok {
						w.Close()
						return
					}
					if w.Write(evt) != nil {
						w.Close()
						return
					}
				default:
					w.Write(protocol.NewEndEvent())
					return
				}
			}
		}
	}
}

// wsWriter WS 连接写入器：gorilla websocket 不允许并发写，订阅泵与主循环控制事件
// 可能同时写同一连接，统一经互斥锁串行化
type wsWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (w *wsWriter) Write(msg *protocol.EventMsg) error {
	if msg == nil || w.conn == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(websocket.TextMessage, []byte(jsonx.ToStr(msg)))
}

// Close 主动关闭连接（泵异常终止：触发前端重连 + attach 恢复）
func (w *wsWriter) Close() {
	if w.conn == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.conn.Close()
}

// parseResumeList 解析并校验中断恢复参数
func parseResumeList(content string) (*[]tools.InterruptResume, error) {
	resumeList, err := jsonx.ToByStr[[]tools.InterruptResume](content)
	if err != nil {
		return nil, fmt.Errorf("parse resume params error: %w", err)
	}
	if resumeList == nil || len(*resumeList) == 0 {
		return nil, errors.New("empty resume params")
	}
	for i := range *resumeList {
		if err := validatorx.Validate(&(*resumeList)[i]); err != nil {
			return nil, fmt.Errorf("invalid resume params: %w", err)
		}
	}
	return resumeList, nil
}
