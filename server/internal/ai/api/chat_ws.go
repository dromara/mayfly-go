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
	"mayfly-go/internal/ai/pkg/headless"
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
	la := contextx.GetLoginAccount(rc.MetaCtx)
	baseCtx := contextx.WithTraceId(context.Background())
	if la != nil {
		baseCtx = contextx.WithLoginAccount(baseCtx, la)
	}
	// 连接级发起者标识：一次连接内不变，send/stop/attach 三个入口共用它做归属比对
	// （身份属于连接而不是属于消息，不在循环里逐条取）
	connUserId := ""
	if la != nil {
		connUserId = fmt.Sprintf("%d", la.Id)
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
			// 显式 stop：取消 turn ctx（Run 阻塞在 goroutine，不影响读取下一条消息）。
			// 按连接归属限定：否则任何客户端知道会话 id 就能掉停别人的运行中 turn
			if !chatRt.Stop(chatReq.ConversationId, connUserId) {
				w.Write(protocol.NewErrorEvent("no running turn to stop", "server"))
			}
			continue
		case form.ChatMsgTypeAttach:
			// 订阅运行中 turn：先回放缓冲快照，再持续订阅实时事件（刷新/重连续流）
			cancelSub()
			curSub = a.attachRunningTurn(w, chatReq.ConversationId, connUserId)
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
		} else if authErr := a.conversationApp.AssertWritable(rc.MetaCtx, convId); authErr != nil {
			// 客户端传来的会话 id 必须经授权：不校则任何登录账号可往任意会话追加消息（写侧越权），
			// 包括把事后对话写进一次运维运行的审计会话里（同时污染该运行的 token 成本统计）
			w.Write(protocol.NewErrorEvent(authErr.Error(), "server"))
			continue
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

		// 会话级互斥注册（Start 时创建 turn ctx 与事件总线；userId 用于运行中列表与归属比对）
		turn, startErr := chatRt.Start(convId, turnId, connUserId, baseCtx)
		if startErr != nil {
			w.Write(protocol.NewErrorEvent(turnStartErrText(baseCtx, startErr), "server"))
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
//
// userId 参与归属判定：不校时任何客户端可按会话 id 订阅别人的实时输出（流式内容越权）。
// 非本人一律回 turn_not_running，不区分「不存在」与「不属于你」，避免把别人会话的运行状态泄露出去
func (a *Ai) attachRunningTurn(w *wsWriter, convId uint64, userId string) *turnSubscription {
	rt := chatRt.GetRunning(convId, userId)
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
	// 运行关联注入：与无头路径同源机制（见 headless.WithRunAssoc）。
	//
	// 此前只有 agent_runner 会注入 RunAssoc，聊天路径登记的东西拿不到来源会话：实测提案的
	// conversation_id 恒为 0，事后无法从提案回溯当时的诊断上下文（回放入口断链）。
	// 聊天路径没有业务运行 id，故只带会话 id
	turnCtx := headless.WithRunAssoc(turn.ctx, &headless.RunAssoc{ConversationId: convId})
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

	// 轮次终态必须在落库前定下来：显式停止要作为一条 turn_notice item 与本轮其余 item 一起写出去。
	// 旧写法是「先 Flush、后算终态、再把停止提示只推给当前客户端」，于是停止这件事零持久化：
	// 刷新或重进会话后，那段在句子中途断开的回答与模型正常答完完全同形（真机实测到截断处留下
	// 未闭合的 `**` 加粗语法，界面上看着像模型自己写崩了）
	finalization := decideTurnFinalization(err)
	if finalization.status == protocol.TurnStatusStopped {
		recorder.CollectTurnNotice(turnId, entity.TurnNoticeKindStopped)
	}

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

	if finalization.publishError {
		publish(protocol.NewErrorEvent(err.Error(), "agent"))
	}

	publish(protocol.NewTurnCompletedEvent(turnId, finalization.status, turnUsage))

	// turn 收尾：终结事件总线（触发订阅泵 drain 剩余事件并写 end）并移出注册表
	turn.bus.Finish()
	chatRt.Remove(convId, turnId)
}

// turnFinalization turn 出错后的收尾决策：turn 终态 + 是否需要向客户端下发 error 事件
type turnFinalization struct {
	status       string
	publishError bool
}

// decideTurnFinalization 把一次 turn 的错误映射成终态，并决定是否额外下发 error 事件。
//
// 为什么剥成纯函数：这几个分支直接决定用户在会话里看到什么（红色故障条、「已停止」提示、还是只有那条
// 收尾气泡），而它的输入只是一个 error，不需要 WS 连接、不需要 LLM。留在 handler 里内联时它不可测，
// 「用户自己取消了补全却被显示成系统故障」这种回归只能靠真机点出来（就是本轮真机发现的那条残留）。
//
// 用户主动终止（取消参数补全 / 拒绝审批）刻意**不**下发 error 事件：那两条文案本是工具层给模型的指令，
// 外层还包着 `[NodeRunError] ... node path: [...]` 这类内部节点结构，而前端红色 notice 的唯一生产者
// 就是 error 事件（useChatCallbacks.onError），推过去必然多亮一行红。收尾文案由 agent 落的 i18n 气泡承担。
// 也不复用 stopped：stopped 表示「用户按了停止、中断了整轮生成」，语义不同，前端据此决定是否写停止提示
func decideTurnFinalization(err error) turnFinalization {
	if err == nil {
		return turnFinalization{status: protocol.TurnStatusSuccess}
	}
	switch {
	case errors.Is(err, context.Canceled):
		return turnFinalization{status: protocol.TurnStatusStopped}
	case agent.IsUserTerminated(err):
		return turnFinalization{status: protocol.TurnStatusCancelled}
	default:
		return turnFinalization{status: protocol.TurnStatusFailed, publishError: true}
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
