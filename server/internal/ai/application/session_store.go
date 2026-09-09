package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/session"
	fileapp "mayfly-go/internal/file/application"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"

	"github.com/cloudwego/eino/schema"
)

// sessionKeyPrefix 会话存储键前缀，sessionKey 格式为 conv:{conversationId}
const sessionKeyPrefix = "conv:"

// sessionStoreImpl 基于 conversation + turn_item 两张表的 session.Store 实现。
//
// LLM 消息不再单独建表：api 层已将每轮产出的消息（user/assistant/tool_call）
// 以 TurnItem 形式持久化到 t_ai_turn_item，本实现在 TurnItem 与 LLM 消息之间双向转换：
//   - 读：GetHistory/GetMessage 将 TurnItem payload 转换为 session.Message
//   - 写：AppendMsgs 无需落库（消息已由 api 层以 TurnItem 形式持久化），
//     消息计数与 token 统计由 Manager 通过 SaveMeta 更新到 conversation 元数据
//   - 中断信息存于 tool_call item 的 extra 列（{"interrupt": ...}），
//     读取时从 extra 派生 internal 消息，恢复流程所需的运行时信息（resumeInfo、
//     补全后的工具参数）通过 UpdateMessage 回写到 item extra/payload
type sessionStoreImpl struct {
	conversationRepo repository.Conversation `inject:"T"`
	turnItemRepo     repository.TurnItem     `inject:"T"`
	// fileApp 文件服务：用户消息 image 段的 fileKey 引用（附件已落 local/S3）
	// 在重建 LLM 消息时解析为 base64 data URL
	fileApp fileapp.File `inject:"T"`
}

var _ session.Store = (*sessionStoreImpl)(nil)

// parseConvId 从 sessionKey（conv:{conversationId}）解析会话 ID
func parseConvId(sessionKey string) (uint64, error) {
	if !strings.HasPrefix(sessionKey, sessionKeyPrefix) {
		return 0, errorx.NewBiz(fmt.Sprintf("invalid session key: %s", sessionKey))
	}
	convId, err := strconv.ParseUint(strings.TrimPrefix(sessionKey, sessionKeyPrefix), 10, 64)
	if err != nil {
		return 0, errorx.NewBiz(fmt.Sprintf("invalid session key: %s", sessionKey))
	}
	return convId, nil
}

// AppendMsgs 会话消息持久化。
// 消息数据本身已由 api 层事件收集以 TurnItem 形式写入 t_ai_turn_item，此处无需重复落库；
// 消息计数与 token 统计由 Manager 在调用后通过 SaveMeta 更新到 conversation 元数据。
func (s *sessionStoreImpl) AppendMsgs(ctx context.Context, sessionKey string, msgs ...*session.Message) error {
	return nil
}

// GetHistory 获取会话历史消息（按时间正序），由 TurnItem 转换而来
func (s *sessionStoreImpl) GetHistory(ctx context.Context, sessionKey string, limit int) ([]*session.Message, error) {
	convId, err := parseConvId(sessionKey)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 1000
	}
	items, err := s.turnItemRepo.SelectRecentByConvId(ctx, convId, limit)
	if err != nil {
		return nil, err
	}
	msgs := make([]*session.Message, 0, len(items))
	for _, item := range items {
		msgs = append(msgs, s.itemToMessages(ctx, item)...)
	}
	// 单个 item 可能产生多条消息（tool_call item → 助手调用 + 工具结果），裁剪为最后 limit 条
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

// ClearHistory 清空会话历史消息（删除 TurnItem，保留会话元数据）
func (s *sessionStoreImpl) ClearHistory(ctx context.Context, sessionKey string) error {
	convId, err := parseConvId(sessionKey)
	if err != nil {
		return err
	}
	return s.turnItemRepo.DeleteByCond(ctx, &entity.TurnItem{ConversationId: convId})
}

// GetMessage 根据查询条件获取消息（toolCallId/turnId 全局唯一，无需会话维度过滤）
func (s *sessionStoreImpl) GetMessage(ctx context.Context, query *session.MessageQuery) ([]*session.Message, error) {
	itemType := itemTypeOfMessageType(query.MessageType)
	cond := model.NewCond()
	if itemType != "" {
		cond = cond.Eq("item_type", itemType)
	} else if query.ActionId != "" {
		// actionId 自含于 item extra 的中断信息（无查询列），收窄到 tool_call 后内存过滤
		cond = cond.Eq("item_type", entity.ItemTypeToolCall)
	}
	if query.TurnId != "" {
		cond = cond.Eq("turn_id", query.TurnId)
	}
	if query.ToolCallId != "" {
		cond = cond.Eq("tool_call_id", query.ToolCallId)
	}
	cond = cond.OrderByDesc("id")

	items, err := s.turnItemRepo.SelectByCond(cond)
	if err != nil {
		return nil, err
	}

	msgs := make([]*session.Message, 0, len(items))
	for _, item := range items {
		for _, msg := range s.itemToMessages(ctx, item) {
			if !matchMessageQuery(msg, query.MessageType) {
				continue
			}
			if query.ActionId != "" && msg.ActionId != query.ActionId {
				continue
			}
			msgs = append(msgs, msg)
		}
	}
	if len(msgs) == 0 {
		return nil, errorx.NewBiz("no message")
	}
	return msgs, nil
}

// UpdateMessage 更新单条消息，将变更回写到对应的 TurnItem：
//   - internal 消息（中断）：resumeInfo 回写到 tool_call item extra 列
//   - tool_call 消息：回填参数补全后的最终工具调用参数到 payload
func (s *sessionStoreImpl) UpdateMessage(ctx context.Context, msg *session.Message) error {
	if msg.Id == 0 {
		return nil
	}
	item, err := s.turnItemRepo.GetById(uint64(msg.Id))
	if err != nil {
		return err
	}

	// 中断消息的 resumeInfo 回写：决策内嵌 interrupt 对象（merge_patch）
	if msg.Role == session.RoleInternal {
		resumeInfo, ok := msg.Extra["resumeInfo"]
		if !ok {
			return nil
		}
		info := interruptInfoOf(item)
		if info == nil {
			return nil
		}
		info.Resume = protocol.NewResumeFromResumeInfo(resumeInfo)
		item.Extra["interrupt"] = info
		return s.turnItemRepo.UpdateById(ctx, item, "extra")
	}

	// 其余仅 tool_call 消息有运行时回写需求：参数补全后的最终工具调用参数回填 payload
	if len(msg.ToolCalls) == 0 {
		return nil
	}
	ti, err := protocol.FromPayload(item.ItemType, item.ItemId, item.Payload)
	if err != nil || ti == nil || ti.Type != protocol.TurnItemTypeToolCall {
		return errorx.NewBiz(fmt.Sprintf("unmarshal turn item payload failed, itemId=%d", item.Id))
	}
	for _, tc := range msg.ToolCalls {
		if tc.ID == ti.ToolCallId {
			ti.Arguments = tc.Function.Arguments
		}
	}
	item.Payload = ti.PayloadJSON()
	return s.turnItemRepo.Save(ctx, item)
}

// ListMetas 列出所有会话元信息（会话列表由 conversation 模块提供，此处不支持）
func (s *sessionStoreImpl) ListMetas(ctx context.Context) ([]*session.SessionMeta, error) {
	return nil, errorx.NewBiz("not implemented")
}

// GetMeta 获取会话元信息（不存在时返回 nil, nil）。
// 元数据映射：Summary→Compaction，Skip→CompactionMessageCount，Count→TotalMessageCount，
// TokenCount→conversation extra 的 contextTokens
func (s *sessionStoreImpl) GetMeta(ctx context.Context, sessionKey string) (*session.SessionMeta, error) {
	convId, err := parseConvId(sessionKey)
	if err != nil {
		return nil, err
	}
	conv := &entity.Conversation{Id: convId}
	if err := s.conversationRepo.GetByCond(conv); err != nil {
		// 会话不存在属正常情况（首次创建前），返回空
		logx.DebugfContext(ctx, "[GetMeta] conversation not found, convId=%d, err=%v", convId, err)
		return nil, nil
	}

	meta := &session.SessionMeta{
		Key:        sessionKey,
		Summary:    conv.Compaction,
		Count:      conv.TotalMessageCount,
		TokenCount: conv.GetExtraInt("contextTokens"),
		Skip:       conv.CompactionMessageCount,
	}
	if conv.CreateTime != nil {
		meta.CreatedAt = *conv.CreateTime
	}
	if conv.UpdateTime != nil {
		meta.UpdatedAt = *conv.UpdateTime
	}
	return meta, nil
}

// SaveMeta 保存会话元信息到 conversation（不存在时兜底创建）
//
// 已存在的会话采用精确列更新而非全行 Save：turn 完成时 token 统计列
// （IncrTokenUsage 原子自增）与本方法并发写入，全行覆盖会丢失增量。
func (s *sessionStoreImpl) SaveMeta(ctx context.Context, meta *session.SessionMeta) error {
	convId, err := parseConvId(meta.Key)
	if err != nil {
		return err
	}
	conv := &entity.Conversation{Id: convId}
	if err := s.conversationRepo.GetByCond(conv); err != nil {
		// 正常流程会话由 api 层先行创建，此处为兜底创建（新建场景全量写入）
		conv = &entity.Conversation{
			Code:   stringx.RandUUID(),
			Title:  meta.Extra.GetStr("title"),
			Status: entity.ConversationStatusActive,
		}
		conv.Id = convId
		conv.Compaction = meta.Summary
		conv.CompactionMessageCount = meta.Skip
		conv.TotalMessageCount = meta.Count
		conv.SetExtraValue("contextTokens", meta.TokenCount)
		return s.conversationRepo.Save(ctx, conv)
	}

	conv.Compaction = meta.Summary
	conv.CompactionMessageCount = meta.Skip
	conv.TotalMessageCount = meta.Count
	conv.SetExtraValue("contextTokens", meta.TokenCount)
	return s.conversationRepo.UpdateById(ctx, conv,
		"compaction", "compaction_message_count", "total_message_count", "extra")
}

// DeleteMeta 删除会话元信息（级联删除会话下的 TurnItem）
func (s *sessionStoreImpl) DeleteMeta(ctx context.Context, sessionKey string) error {
	convId, err := parseConvId(sessionKey)
	if err != nil {
		return err
	}
	if err := s.turnItemRepo.DeleteByCond(ctx, &entity.TurnItem{ConversationId: convId}); err != nil {
		return err
	}
	return s.conversationRepo.DeleteByCond(ctx, &entity.Conversation{Id: convId})
}

// itemTypeOfMessageType 将 LLM 消息类型映射为 TurnItem 类型（"" 表示不过滤）。
// 中断类消息存于 tool_call item extra 列，同样映射到 tool_call
func itemTypeOfMessageType(messageType string) string {
	switch messageType {
	case "":
		return ""
	case entity.MsgTypeUser, entity.MsgTypeAssistant:
		return entity.ItemTypeMessage
	default:
		// tool_call/tool_result 及中断类型（interrupt_approval / interrupt_param_completion ...）
		return entity.ItemTypeToolCall
	}
}

// matchMessageQuery 内存精确匹配消息类型（item_type 列过滤是粗粒度的，需二次过滤）
func matchMessageQuery(msg *session.Message, messageType string) bool {
	if messageType == "" {
		return true
	}
	switch messageType {
	case entity.MsgTypeToolCall:
		return len(msg.ToolCalls) > 0
	case entity.MsgTypeToolResult:
		return msg.Role == schema.Tool
	case entity.MsgTypeUser, entity.MsgTypeAssistant:
		return string(msg.Role) == messageType
	default:
		// internal 消息按 extra.type 匹配（中断类型），"internal" 匹配所有内部消息
		if msg.Role != session.RoleInternal {
			return false
		}
		t := msg.Extra.GetStr("type")
		return t == messageType || t == ""
	}
}

// itemToMessages 将 TurnItem 转换为 LLM 消息（一条 item 可能对应多条消息，reasoning 不参与上下文）。
// image 段的 fileKey 引用经文件服务解析为 base64 data URL（旧数据 data URL 直接透传）
func (s *sessionStoreImpl) itemToMessages(ctx context.Context, item *entity.TurnItem) []*session.Message {
	ti, err := protocol.FromPayload(item.ItemType, item.ItemId, item.Payload)
	if err != nil || ti == nil {
		logx.Warnf("[itemToMessages] unmarshal turn item payload failed, itemId=%d, err=%v", item.Id, err)
		return nil
	}

	switch ti.Type {
	case protocol.TurnItemTypeMessage:
		// 普通消息：拼接文本分段
		var sb strings.Builder
		for _, seg := range ti.Content {
			sb.WriteString(seg.Text)
		}
		return []*session.Message{{
			Id:        int64(item.Id),
			TurnId:    item.TurnId,
			Role:      schema.RoleType(ti.Role),
			MsgType:   ti.Role,
			Content:   sb.String(),
			ImageUrls: ResolveImageUrls(ctx, s.fileApp, protocol.ImageUrlsOf(ti.Content)),
		}}

	case protocol.TurnItemTypeToolCall:
		// 工具调用 item → 助手调用消息 + 工具结果消息
		if ti.ToolCallId == "" {
			return nil
		}
		// 防御：历史数据可能存在 arguments 为空的 tool_call（旧版本落库缺陷），
		// 直接发给 LLM 会被网关拒绝（function.arguments must be defined），补空 JSON 对象自愈
		args := ti.Arguments
		if args == "" {
			args = "{}"
		}
		msgs := []*session.Message{
			{
				Id:      int64(item.Id),
				TurnId:  item.TurnId,
				Role:    schema.Assistant,
				MsgType: entity.MsgTypeToolCall,
				ToolCalls: []schema.ToolCall{{
					ID:       ti.ToolCallId,
					Function: schema.FunctionCall{Name: ti.ToolName, Arguments: args},
				}},
			},
			{
				Id:         int64(item.Id),
				TurnId:     item.TurnId,
				Role:       schema.Tool,
				MsgType:    entity.MsgTypeToolResult,
				Content:    ti.Output,
				ToolCallId: ti.ToolCallId,
				ToolName:   ti.ToolName,
			},
		}
		// 中断信息存于 tool_call item extra 列（{"interrupt": InterruptInfo}），
		// 派生 internal 消息供中断恢复流程读取（与 internal item 消息同构）
		if info := interruptInfoOf(item); info != nil {
			msgType := protocol.InterruptMsgType(info.Kind)
			extra := collx.M{"type": msgType}
			// 恢复决策内嵌 interrupt 对象，反合成为恢复链路消费的 resumeInfo
			if info.Resume != nil {
				extra["resumeInfo"] = info.Resume.ToResumeInfo(info.Kind, info.RequestId, item.TurnId)
			}
			msgs = append(msgs, &session.Message{
				Id:         int64(item.Id),
				TurnId:     item.TurnId,
				Role:       session.RoleInternal,
				MsgType:    msgType,
				Content:    info.Message,
				ToolCallId: ti.ToolCallId,
				ToolName:   ti.ToolName,
				ActionId:   info.RequestId,
				Extra:      extra,
			})
		}
		return msgs
	}

	// reasoning/compaction 等不参与 LLM 上下文的 item
	return nil
}

// interruptInfoOf 从 tool_call item 的 extra 列提取中断信息
// （extra["interrupt"] 为 InterruptInfo 强类型结构，内存态为结构体，
// 经 extra 列 JSON 落库后为 map，两种形态均需兼容）
func interruptInfoOf(item *entity.TurnItem) *protocol.InterruptInfo {
	if item.Extra == nil {
		return nil
	}
	switch v := item.Extra["interrupt"].(type) {
	case *protocol.InterruptInfo:
		return v
	case protocol.InterruptInfo:
		return &v
	default:
		if v == nil {
			return nil
		}
		info, err := jsonx.ToByStr[protocol.InterruptInfo](jsonx.ToStr(v))
		if err != nil || info == nil || info.Kind == "" {
			return nil
		}
		return info
	}
}
