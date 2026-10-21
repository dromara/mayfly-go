package application

import (
	"context"
	"mayfly-go/internal/ai/application/dto"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

// Conversation 会话管理服务
type Conversation interface {
	base.App[*entity.Conversation]

	// ListConversations 列出用户会话列表
	ListConversations(ctx context.Context, query *dto.ConversationQuery) ([]*entity.Conversation, error)

	// AssertReadable 校验当前登录账号能否读取该会话的消息内容（按 id 取消息是内容级越权面）：
	//   - 用户发起的会话（source=user）仅创建者本人可读
	//   - 系统任务触发的会话（source=system）对所有登录账号可读：它承载的是运维运行结论，
	//     与按平台惯例不带权限码的 /agentops/runs 列表属同一可见面，执行历史回放必须能直达
	//
	// 会话列表接口本来就走 creatorId 过滤，但消息详情接口按路径 id 直取不收 creator 约束，
	// 两者口径不一致时“列表看不到”并不等于“数据拿不到”；新增的「执行历史 → 按 convId 回放」
	// 入口又使这个差异有了可直接点击的路径，故在此把规则收敛成一个判定点
	AssertReadable(ctx context.Context, convId uint64) error

	// AssertOwned 校验会话归属于当前登录账号（改名/删除等写操作用）。
	// 与 AssertReadable 分开：系统会话对所有登录账号可读（运维回放），但不得被任意账号改名/删除
	AssertOwned(ctx context.Context, convId uint64) error

	// AssertWritable 校验当前登录账号能否在该会话发起新一轮对话（写入消息/工具调用/token 统计）。
	// 比 AssertOwned 多一道闸：系统运行会话（source=system）一律不可写。
	//
	// 为什么必须单独一个判定点：这类会话是一次运维运行的审计产物，写进去会同时破坏两件事——
	// 结论不再可回放（混入事后对话），以及运行成本被改写（IncrTokenUsage 会累加到同一条统计）。
	// 系统会话的 creator 恰好是运行 owner，故不能复用 AssertOwned（owner 会恰好通过）
	AssertWritable(ctx context.Context, convId uint64) error

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

// authConversation 三个授权判定的共同前置：必须有登录态、会话必须存在。
//
// 收敛成一处的原因：登录态缺失/会话不存在这两条判据若分散在各 Assert* 里，很容易出现
// 某个分支漏判（无登录态被当成放行）或错误文案与实际判据不一致，而这类偏差在接口上是看不出来的
func (c *conversationAppImpl) authConversation(ctx context.Context, convId uint64) (*entity.Conversation, *model.LoginAccount, error) {
	la := contextx.GetLoginAccount(ctx)
	if la == nil || la.Id == 0 {
		return nil, nil, errorx.NewBizI(ctx, imsg.ErrConvLoginRequired)
	}
	conv, err := c.GetById(convId, "id", "creatorId", "source")
	if err != nil || conv == nil {
		return nil, nil, errorx.NewBizI(ctx, imsg.ErrConvNotFound)
	}
	return conv, la, nil
}

// AssertReadable 会话消息读取授权（规则见接口注释）
func (c *conversationAppImpl) AssertReadable(ctx context.Context, convId uint64) error {
	conv, la, err := c.authConversation(ctx, convId)
	if err != nil {
		return err
	}
	if conv.Source == entity.ConversationSourceSystem {
		return nil
	}
	if conv.CreatorId != la.Id {
		return errorx.NewBizI(ctx, imsg.ErrConvReadDenied)
	}
	return nil
}

// AssertOwned 会话写操作归属校验（规则见接口注释）
func (c *conversationAppImpl) AssertOwned(ctx context.Context, convId uint64) error {
	conv, la, err := c.authConversation(ctx, convId)
	if err != nil {
		return err
	}
	if conv.CreatorId != la.Id {
		return errorx.NewBizI(ctx, imsg.ErrConvModifyDenied)
	}
	return nil
}

// AssertWritable 会话可发起新对话的授权校验（规则见接口注释）
func (c *conversationAppImpl) AssertWritable(ctx context.Context, convId uint64) error {
	conv, la, err := c.authConversation(ctx, convId)
	if err != nil {
		return err
	}
	if conv.Source == entity.ConversationSourceSystem {
		return errorx.NewBizI(ctx, imsg.ErrConvSystemReadOnly)
	}
	if conv.CreatorId != la.Id {
		return errorx.NewBizI(ctx, imsg.ErrConvWriteDenied)
	}
	return nil
}

func (c *conversationAppImpl) ListConversations(ctx context.Context, query *dto.ConversationQuery) ([]*entity.Conversation, error) {
	cond := model.NewCond().
		Eq("creatorId", query.UserId).
		// 系统任务触发的会话（agentops 运行/编排/评测）不进用户会话列表，
		// 与用户发起的 chat 区分；回放经执行历史按 convId 直达
		Ne("source", entity.ConversationSourceSystem).
		Eq("status", entity.ConversationStatusActive).
		OrderByDesc("id")
	return c.ListByCond(cond)
}

func (c *conversationAppImpl) CreateConversation(ctx context.Context, req *dto.CreateConversationReq) (*entity.Conversation, error) {
	source := req.Source
	if source == "" {
		source = entity.ConversationSourceUser
	}
	conv := &entity.Conversation{
		Code:   stringx.RandUUID(),
		Title:  req.Title,
		Status: entity.ConversationStatusActive,
		Source: source,
	}
	if conv.Title == "" {
		// 哨兵值取中立包的同一常量：前端靠它识别「尚未命名」并用首条消息自动改名，
		// 两处字面量各自演化会让自动命名静默失效（标题永远停在默认值）
		conv.Title = headless.DefaultConversationTitle
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
	// 累加走数据库侧原子自增（Patch.Inc）：没有读-改-写窗口，也不需要应用层手拼 UPDATE 语句
	patch := base.NewPatch().
		Inc("prompt_tokens", promptTokens).
		Inc("completion_tokens", completionTokens).
		Inc("total_tokens", totalTokens)
	// 走应用层入口（只报执行是否出错）：这里不关心影响行数，会话不存在时 0 行属正常态
	return c.UpdateColsByCond(ctx, patch, model.NewCond().Eq(model.IdColumn, convId))
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
