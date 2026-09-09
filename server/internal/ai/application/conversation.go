package application

import (
	"context"
	"mayfly-go/internal/ai/application/dto"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

// Conversation 会话管理服务
type Conversation interface {
	base.App[*entity.Conversation]

	// ListConversations 列出用户会话列表
	ListConversations(ctx context.Context, query *dto.ConversationQuery) ([]*entity.Conversation, error)

	// CreateConversation 创建新会话
	CreateConversation(ctx context.Context, req *dto.CreateConversationReq) (*entity.Conversation, error)

	// DeleteConversation 删除会话
	DeleteConversation(ctx context.Context, id uint64) error

	// UpdateTitle 更新会话标题
	UpdateTitle(ctx context.Context, id uint64, title string) error

	// LoadTurnGroups 按 turn 分组加载消息（分页）
	LoadTurnGroups(ctx context.Context, convId uint64, beforeTurnId string, limit int) ([]*dto.TurnGroupDTO, error)

	// GetItemsByTurnId 按 turn 获取 item 详情列表
	GetItemsByTurnId(ctx context.Context, convId uint64, turnId string) ([]*dto.TurnItemDTO, error)

	// AppendTurnItems 追加 turn item 到会话
	AppendTurnItems(ctx context.Context, items ...*entity.TurnItem) error

	// GetByCode 根据编码获取会话
	GetByCode(ctx context.Context, code string) (*entity.Conversation, error)

	// IncrTokenUsage 累加会话 token 用量统计（turn 完成时由 api 层调用）
	IncrTokenUsage(ctx context.Context, convId uint64, promptTokens, completionTokens, totalTokens int64) error
}

type conversationAppImpl struct {
	base.AppImpl[*entity.Conversation, repository.Conversation]

	turnItemRepo repository.TurnItem `inject:"T"`
}

var _ Conversation = (*conversationAppImpl)(nil)

func (c *conversationAppImpl) ListConversations(ctx context.Context, query *dto.ConversationQuery) ([]*entity.Conversation, error) {
	cond := model.NewCond().
		Eq("creatorId", query.UserId).
		Eq("status", entity.ConversationStatusActive).
		OrderByDesc("id")
	return c.ListByCond(cond)
}

func (c *conversationAppImpl) CreateConversation(ctx context.Context, req *dto.CreateConversationReq) (*entity.Conversation, error) {
	conv := &entity.Conversation{
		Code:   stringx.RandUUID(),
		Title:  req.Title,
		Status: entity.ConversationStatusActive,
	}
	if conv.Title == "" {
		conv.Title = "New Chat"
	}
	err := c.Save(ctx, conv)
	if err != nil {
		return nil, err
	}
	return conv, nil
}

func (c *conversationAppImpl) DeleteConversation(ctx context.Context, id uint64) error {
	// 级联删除会话下的 turn items（含 LLM 上下文消息）
	if err := c.turnItemRepo.DeleteByCond(ctx, &entity.TurnItem{ConversationId: id}); err != nil {
		return err
	}
	return c.DeleteById(ctx, id)
}

func (c *conversationAppImpl) UpdateTitle(ctx context.Context, id uint64, title string) error {
	return c.UpdateByCond(ctx, &entity.Conversation{Title: title}, &entity.Conversation{Id: id})
}

func (c *conversationAppImpl) LoadTurnGroups(ctx context.Context, convId uint64, beforeTurnId string, limit int) ([]*dto.TurnGroupDTO, error) {
	if limit <= 0 {
		limit = 20
	}

	groups, err := c.turnItemRepo.SelectTurnGroups(ctx, convId, beforeTurnId, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.TurnGroupDTO, 0, len(groups))
	for _, g := range groups {
		items := make([]*dto.TurnItemDTO, 0, len(g.Items))
		for _, item := range g.Items {
			items = append(items, toTurnItemDTO(item))
		}
		result = append(result, &dto.TurnGroupDTO{
			TurnId: g.TurnId,
			Items:  items,
		})
	}

	return result, nil
}

func (c *conversationAppImpl) GetItemsByTurnId(ctx context.Context, convId uint64, turnId string) ([]*dto.TurnItemDTO, error) {
	items, err := c.turnItemRepo.SelectByTurnId(ctx, convId, turnId)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.TurnItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toTurnItemDTO(item))
	}
	return result, nil
}

func (c *conversationAppImpl) AppendTurnItems(ctx context.Context, items ...*entity.TurnItem) error {
	if len(items) == 0 {
		return nil
	}
	return c.turnItemRepo.BatchInsert(ctx, items)
}

func (c *conversationAppImpl) GetByCode(ctx context.Context, code string) (*entity.Conversation, error) {
	conv := &entity.Conversation{Code: code}
	err := c.GetByCond(conv)
	if err != nil {
		return nil, err
	}
	return conv, nil
}

// IncrTokenUsage 累加会话 token 用量统计（turn 完成时由 api 层调用）
//
// 原子自增：无读改写，单条 UPDATE 完成，与 SaveMeta 的元数据列更新互不干扰。
func (c *conversationAppImpl) IncrTokenUsage(ctx context.Context, convId uint64, promptTokens, completionTokens, totalTokens int64) error {
	table := new(entity.Conversation).TableName()
	sql := "UPDATE " + table +
		" SET prompt_tokens = prompt_tokens + ?, completion_tokens = completion_tokens + ?, total_tokens = total_tokens + ?" +
		" WHERE id = ?"
	return c.Repo.ExecBySql(sql, promptTokens, completionTokens, totalTokens, convId)
}

// toTurnItemDTO 将 entity.TurnItem 转换为 dto.TurnItemDTO
// （payload 剥离了 type/id，由 item_type / item_id 列回填，TurnItemRow::to_turn_item）
func toTurnItemDTO(item *entity.TurnItem) *dto.TurnItemDTO {
	dto := &dto.TurnItemDTO{
		Id:             item.Id,
		ConversationId: item.ConversationId,
		TurnId:         item.TurnId,
		ItemType:       item.ItemType,
		ItemId:         item.ItemId,
		Status:         item.Status,
		ToolCallId:     item.ToolCallId,
		Extra:          item.Extra,
		CreateTime:     item.CreateTime,
	}
	if item.Payload != "" {
		turnItem, err := protocol.FromPayload(item.ItemType, item.ItemId, item.Payload)
		if err != nil {
			logx.Warnf("[toTurnItemDTO] unmarshal turn item payload failed, itemId=%d, err=%v", item.Id, err)
		}
		if turnItem != nil {
			dto.Item = turnItem
		}
	}
	return dto
}
