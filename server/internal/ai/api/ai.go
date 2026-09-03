package api

import (
	"fmt"
	"mayfly-go/internal/ai/api/form"
	"mayfly-go/internal/ai/api/vo"
	"mayfly-go/internal/ai/application"
	"mayfly-go/internal/ai/application/dto"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/skill"
	dbapp "mayfly-go/internal/db/application"
	meentity "mayfly-go/internal/db/domain/entity"
	machineapp "mayfly-go/internal/machine/application"
	mcentity "mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strconv"
	"time"
)

// Ai API 结构体，处理 AI 相关请求
type Ai struct {
	conversationApp application.Conversation `inject:"T"`
	turnItemApp     application.TurnItem     `inject:"T"`
	machineApp      machineapp.Machine       `inject:"T"`
	dbApp           dbapp.Db                 `inject:"T"`
}

// ReqConfs 获取 AI 相关的请求配置
func (a *Ai) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 会话管理
		req.NewGet("/chat/conversations", a.ChatConversations),
		req.NewPost("/chat/conversations", a.CreateConversation),
		req.NewDelete("/chat/conversations/:id", a.DeleteConversation),
		req.NewPost("/chat/conversations/rename", a.RenameConversation),
		// TurnItem 查询
		req.NewGet("/chat/conversations/:id/items", a.ChatTurnItems),
		req.NewGet("/chat/conversations/:id/items/:turnId", a.ChatItemsByTurn),
		// 技能/资源引用选择（`/` `@` 触发菜单数据源）
		req.NewGet("/chat/skills", a.ChatSkills),
		req.NewGet("/chat/resources", a.ChatResources),
		// 运行中 turn 列表（会话列表执行中指示器数据源）
		req.NewGet("/chat/running-turns", a.ChatRunningTurns),
		// WebSocket 聊天（EventMsg 协议）
		req.NewGet("/chat/ws", a.Chat).NoRes(),
	}
	return req.NewConfs("/ai", reqs[:]...)
}

// ============== 会话管理 ==============

// ChatConversations 获取会话列表
func (a *Ai) ChatConversations(rc *req.Ctx) {
	convs, err := a.conversationApp.ListConversations(rc.MetaCtx, &dto.ConversationQuery{
		UserId: rc.GetLoginAccount().Id,
	})
	biz.ErrIsNil(err)
	rc.ResData = collx.ArrayMap(convs, func(c *entity.Conversation) *vo.ConversationVO {
		return &vo.ConversationVO{
			Id:               c.Id,
			Code:             c.Code,
			Title:            c.Title,
			Status:           c.Status,
			MessageCount:     c.TotalMessageCount,
			PromptTokens:     c.PromptTokens,
			CompletionTokens: c.CompletionTokens,
			TotalTokens:      c.TotalTokens,
			CreateTime:       formatTime(c.CreateTime),
			UpdateTime:       formatTime(c.UpdateTime),
		}
	})
}

// CreateConversation 创建会话
func (a *Ai) CreateConversation(rc *req.Ctx) {
	reqBody := rc.BindJson[form.CreateConversationRequest]()
	conv, err := a.conversationApp.CreateConversation(rc.MetaCtx, &dto.CreateConversationReq{
		Title: reqBody.Title,
	})
	biz.ErrIsNil(err)
	rc.ResData = &vo.ConversationVO{
		Id:         conv.Id,
		Code:       conv.Code,
		Title:      conv.Title,
		Status:     conv.Status,
		CreateTime: formatTime(conv.CreateTime),
		UpdateTime: formatTime(conv.UpdateTime),
	}
}

// DeleteConversation 删除会话
func (a *Ai) DeleteConversation(rc *req.Ctx) {
	id, err := strconv.ParseUint(rc.PathParam("id"), 10, 64)
	biz.ErrIsNil(err)
	biz.ErrIsNil(a.conversationApp.DeleteConversation(rc.MetaCtx, id))
}

// RenameConversation 重命名会话
func (a *Ai) RenameConversation(rc *req.Ctx) {
	rename := rc.BindJson[form.RenameConversationRequest]()
	biz.ErrIsNil(a.conversationApp.UpdateTitle(rc.MetaCtx, rename.Id, rename.Title))
}

// ChatRunningTurns 获取当前用户运行中的 turn 列表（进入 AI 页面时校正会话列表执行中状态）
func (a *Ai) ChatRunningTurns(rc *req.Ctx) {
	rc.ResData = chatRt.ListRunning(fmt.Sprintf("%d", rc.GetLoginAccount().Id))
}

// ============== 引用选择（技能 / 资源） ==============

// ChatSkills 可引用技能列表（`/` 触发菜单数据源）
func (a *Ai) ChatSkills(rc *req.Ctx) {
	skills := skill.DefaultRegistry.List()
	rc.ResData = collx.ArrayMap(skills, func(s *skill.Skill) *vo.ChatSkillVO {
		return &vo.ChatSkillVO{
			Id:          s.Code,
			Name:        s.Name,
			Description: s.Description,
		}
	})
}

// ChatResources 可引用资源列表（`@` 触发菜单数据源，随消息下发使模型明确目标资源，
// 避免工具调用因资源参数缺失而中断等待用户手动补充）
func (a *Ai) ChatResources(rc *req.Ctx) {
	resources := make([]*vo.ChatResourceVO, 0)

	machines, err := a.machineApp.ListByCond(&mcentity.Machine{
		Status:   mcentity.MachineStatusEnable,
		Protocol: mcentity.MachineProtocolSsh,
	}, "id", "code", "name", "ip", "port")
	biz.ErrIsNil(err)
	for _, m := range machines {
		resources = append(resources, &vo.ChatResourceVO{
			Id:           strconv.FormatUint(m.Id, 10),
			ResourceType: "machine",
			Name:         m.Name,
			Code:         m.Code,
			Ip:           m.Ip,
			Port:         m.Port,
			Description:  fmt.Sprintf("%s:%d", m.Ip, m.Port),
		})
	}

	dbs, err := a.dbApp.ListByCond(&meentity.Db{}, "id", "code", "name")
	biz.ErrIsNil(err)
	for _, d := range dbs {
		resources = append(resources, &vo.ChatResourceVO{
			Id:           strconv.FormatUint(d.Id, 10),
			ResourceType: "db",
			Name:         d.Name,
			Code:         d.Code,
			Description:  d.Code,
		})
	}

	rc.ResData = resources
}

// ============== TurnItem 查询 ==============

// ChatTurnItems 按 turn 分组加载消息（分页）
func (a *Ai) ChatTurnItems(rc *req.Ctx) {
	convId, err := strconv.ParseUint(rc.PathParam("id"), 10, 64)
	biz.ErrIsNil(err)
	beforeTurnId := rc.Query("beforeTurnId")
	limit := rc.QueryIntDefault("limit", 20)

	groups, err := a.conversationApp.LoadTurnGroups(rc.MetaCtx, convId, beforeTurnId, limit)
	biz.ErrIsNil(err)
	rc.ResData = collx.ArrayMap(groups, func(g *dto.TurnGroupDTO) *vo.TurnGroupVO {
		return &vo.TurnGroupVO{
			TurnId: g.TurnId,
			Items: collx.ArrayMap(g.Items, func(item *dto.TurnItemDTO) *vo.TurnItemVO {
				return &vo.TurnItemVO{
					Id:         item.Id,
					TurnId:     item.TurnId,
					ItemType:   item.ItemType,
					ItemId:     item.ItemId,
					Item:       item.Item,
					Status:     item.Status,
					ToolCallId: item.ToolCallId,
					Extra:      item.Extra,
					CreateTime: item.CreateTime,
				}
			}),
		}
	})
}

// ChatItemsByTurn 按 turnId 加载单个 turn 的 items
func (a *Ai) ChatItemsByTurn(rc *req.Ctx) {
	convId, err := strconv.ParseUint(rc.PathParam("id"), 10, 64)
	biz.ErrIsNil(err)
	turnId := rc.PathParam("turnId")

	items, err := a.conversationApp.GetItemsByTurnId(rc.MetaCtx, convId, turnId)
	biz.ErrIsNil(err)
	rc.ResData = collx.ArrayMap(items, func(item *dto.TurnItemDTO) *vo.TurnItemVO {
		return &vo.TurnItemVO{
			Id:         item.Id,
			TurnId:     item.TurnId,
			ItemType:   item.ItemType,
			ItemId:     item.ItemId,
			Item:       item.Item,
			Status:     item.Status,
			ToolCallId: item.ToolCallId,
			CreateTime: item.CreateTime,
		}
	})
}

// ============== 工具函数 ==============

// buildUserSegments 将前端发送的富文本段转为协议结构化内容段（对齐 tokhub：
// 芯片引用以 typed segment 贯穿发送/持久化/回显，前端按 type 渲染芯片样式）；
// 无段时回落为单条纯文本段
func buildUserSegments(content string, segments []form.ChatSegment) []protocol.ContentSegment {
	if len(segments) == 0 {
		return []protocol.ContentSegment{protocol.NewInputTextSegment(content)}
	}
	result := make([]protocol.ContentSegment, 0, len(segments))
	for _, seg := range segments {
		result = append(result, protocol.ContentSegment{
			Type:  seg.Type,
			Text:  seg.Text,
			Extra: seg.Extra,
		})
	}
	return result
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// toEntityTurnItem 将 protocol.TurnItem 转换为 entity.TurnItem
// （payload 剥离 type/id，由 item_type / item_id 列承载，对齐 tokhub payload_json）
func toEntityTurnItem(convId uint64, turnId string, item *protocol.TurnItem, status string) *entity.TurnItem {
	et := &entity.TurnItem{
		ConversationId: convId,
		TurnId:         turnId,
		ItemId:         item.Id,
		ItemType:       item.Type,
		Payload:        item.PayloadJSON(),
		Status:         status,
		ToolCallId:     item.ToolCallId,
	}
	return et
}

// toolCallItemStatus 从 TurnItem 提取工具调用的实际状态（取值统一用 entity.ItemStatus*）
func toolCallItemStatus(item *protocol.TurnItem) string {
	switch item.Status {
	case protocol.TurnItemStatusFailed:
		return entity.ItemStatusFailed
	case protocol.TurnItemStatusInterrupted:
		return entity.ItemStatusInterrupted
	case protocol.TurnItemStatusSuccess:
		return entity.ItemStatusSuccess
	case protocol.TurnItemStatusCancelled:
		// 恢复路径用户拒绝的真实终态（对齐 tokhub 拒绝 → Cancelled）
		return entity.ItemStatusCancelled
	default:
		return entity.ItemStatusActive
	}
}

// deduplicateTurnItems 按 ItemId 去重，保留每个 ItemId 的最后一条记录
func deduplicateTurnItems(items []*entity.TurnItem) []*entity.TurnItem {
	seen := make(map[string]int) // ItemId -> index in result
	result := make([]*entity.TurnItem, 0, len(items))
	for _, item := range items {
		if idx, ok := seen[item.ItemId]; ok {
			result[idx] = item // 替换为更新的版本
		} else {
			seen[item.ItemId] = len(result)
			result = append(result, item)
		}
	}
	return result
}
