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
	"mayfly-go/internal/ai/application"
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

	"github.com/cloudwego/eino/schema"
	"github.com/gorilla/websocket"
)

// chatRt 会话级运行中 turn 注册表（api 包级单例）
var chatRt = newChatRuntime()

// Chat WebSocket 聊天，使用 EventMsg 结构化事件协议。
//
// turn 运行与 WS 连接解耦（turn registry + replay）：
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

		// 轮次记录器：事件映射下发 + TurnItem 收集与收尾持久化统一由其承担
		// （api 层仅保留 WS 传输职责）
		recorder := application.NewTurnRecorder(a.turnItemApp, convId, turnId, turn.Publish)

		var userMessage []*session.Message
		if resumeParams == nil {
			// 资源/技能芯片引用渲染为文本标记随消息下发，使模型明确目标资源，
			// 避免工具调用因资源参数缺失而中断等待用户手动补充；
			// 渲染经 protocol 注册式分发（新增芯片/资源类型零修改本层）。
			// 保存用户消息为 TurnItem：content 用结构化 segments（芯片引用保留元数据，
			// 历史回显可恢复芯片样式，ContentSegment 贯穿设计）；
			// 发给 LLM 的完整定位标识由 RenderSegments 注入，两者职责分离
			userSegments := buildUserSegments(chatReq.Content, chatReq.Segments)
			// image 段引用提取到 ImageUrls（fileKey 经文件服务解析为 base64 data URL，
			// 旧数据 data URL 直接透传；转 AgenticMessage 时生成 UserInputImage block），
			// 文本侧由 RenderSegments 保留占位说明
			userMessage = collx.AsArray(&session.Message{
				Role:      schema.User,
				Content:   protocol.RenderSegments(userSegments),
				ImageUrls: application.ResolveImageUrls(baseCtx, a.fileApp, protocol.ImageUrlsOf(userSegments)),
			})
			userItem := protocol.NewMessageTurnItem(stringx.SortableUUID(), string(schema.User), userSegments)
			// 附件元数据随 payload 持久化（fileKey 引用，历史回显卡片/预览；展示职责，不参与 LLM 输入）
			userItem.Attachments = buildUserAttachments(chatReq.Attachments)
			recorder.CollectProtocolItem(turnId, userItem, entity.ItemStatusSuccess)
		}

		// turn_started 进入事件总线（发起连接经回放接收，其它订阅者实时接收）
		turn.Publish(protocol.NewTurnStartedEvent(turnId, convId))

		// 独立 goroutine 执行：连接断开/页面刷新不影响 turn 运行
		go a.runTurn(turn, ag, convId, turnId, resumeParams, userMessage, recorder)

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

// runTurn 执行一次 agent turn（独立 goroutine）：事件经 TurnRecorder 映射后发布到
// turn 事件总线（不直接触碰 WS 连接），TurnItem 收集与收尾持久化同样由其承担
func (a *Ai) runTurn(turn *runningTurn, ag *agent.Agent, convId uint64, turnId string, resumeParams []any, userMessage []*session.Message, recorder *application.TurnRecorder) {
	turnCtx := turn.ctx
	publish := turn.Publish

	// 恢复路径预加载：将该 turn 挂起的 interrupted tool_call item
	// 注册到 EventMapper（工具完成时复用原 item_id），并建立 itemId → 恢复决策映射
	if len(resumeParams) > 0 {
		recorder.TrackResumedToolCalls(turnCtx, resumeParams)
	}

	agentRunOptions := []agent.RunOption{
		agent.WithRunSessionKey(fmt.Sprintf("conv:%d", convId)),
		agent.WithTurnId(turnId),
		// 消息映射与 TurnItem 收集统一由 TurnRecorder 承担（api 层仅保留传输职责）
		agent.WithOnChunk(recorder.OnChunk),
		agent.WithOnEvent(recorder.OnEvent),
	}

	if resumeParams != nil {
		agentRunOptions = append(agentRunOptions, agent.WithResumeParams(resumeParams...))
	}

	runRes, err := ag.Run(turnCtx, userMessage, agentRunOptions...)

	// 完成流式输出，收集最终的 TurnItem
	recorder.FinishStreaming()

	// 收尾持久化与统计累计使用无取消 ctx：turn ctx 可能已被显式 stop 取消，仍需保证落库
	saveCtx := context.WithoutCancel(turnCtx)
	recorder.Flush(saveCtx)

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
