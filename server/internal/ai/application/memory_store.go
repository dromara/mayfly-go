package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/memory"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
)

// memoryStoreImpl 基于 t_ai_memory 表的 memory.Store 实现（多实例共享后端）。
//
// 对齐 session_storeImpl 的装配模式：由 application.Init 注册并赋值
// memory.DefaultStore，agent 装配时优先采用；LLM 提取的记忆、记忆工具
// 读写均落 DB，跨会话、跨实例共享。
type memoryStoreImpl struct {
	memoryRepo repository.Memory `inject:"T"`
}

var _ memory.Store = (*memoryStoreImpl)(nil)

// GetByUser 根据用户与标签获取记忆（按 update_time 倒序）
func (s *memoryStoreImpl) GetByUser(ctx context.Context, userID string, tags []string) ([]*memory.MemoryItem, error) {
	memories, err := s.memoryRepo.SelectByUser(ctx, userID, nil, tags, 0)
	if err != nil {
		return nil, err
	}
	items := make([]*memory.MemoryItem, 0, len(memories))
	for _, m := range memories {
		items = append(items, memoryToItem(m))
	}
	return items, nil
}

// Save 保存记忆：带 ID 视为更新（覆盖内容字段），否则新增
func (s *memoryStoreImpl) Save(ctx context.Context, items []*memory.MemoryItem) error {
	for _, item := range items {
		if item == nil {
			continue
		}
		mem, err := itemToMemory(item)
		if err != nil {
			return err
		}
		if mem.Id != 0 {
			if err := s.memoryRepo.UpdateById(ctx, mem,
				"user_id", "type", "content", "tags", "extra"); err != nil {
				return err
			}
		} else if err := s.memoryRepo.Insert(ctx, mem); err != nil {
			return err
		}
	}
	return nil
}

// Delete 删除用户指定 ID 的记忆（限定 user_id，防跨用户删除）
func (s *memoryStoreImpl) Delete(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	ids64 := make([]uint64, 0, len(ids))
	for _, id := range ids {
		id64, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid memory id: %s", id)
		}
		ids64 = append(ids64, id64)
	}
	cond := model.NewCond().Eq("user_id", userID).In("id", ids64)
	return s.memoryRepo.DeleteByCond(ctx, cond)
}

// Search 搜索记忆：query 非空时按关键词模糊匹配（空格分词，任一命中即返回），
// 否则返回最近记忆（对齐 JSONL 存储语义）；后续可叠加 embedding 语义检索
func (s *memoryStoreImpl) Search(ctx context.Context, userID string, query string, limit int) ([]*memory.MemoryItem, error) {
	memories, err := s.memoryRepo.SelectByUser(ctx, userID, strings.Fields(query), nil, limit)
	if err != nil {
		return nil, err
	}
	items := make([]*memory.MemoryItem, 0, len(memories))
	for _, m := range memories {
		items = append(items, memoryToItem(m))
	}
	return items, nil
}

// itemToMemory 领域记忆项 → 表实体（ID 可解析时视为既有记忆，否则新增）
func itemToMemory(item *memory.MemoryItem) (*entity.Memory, error) {
	mem := &entity.Memory{
		UserId:  item.UserID,
		Type:    item.Type,
		Content: item.Content,
	}
	if item.Tags != nil {
		mem.Tags = jsonx.ToStr(item.Tags)
	}
	for k, v := range item.Metadata {
		mem.SetExtraValue(k, v)
	}

	if item.ID != "" {
		id, err := strconv.ParseUint(item.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid memory id: %s", item.ID)
		}
		mem.Id = id
	}
	return mem, nil
}

// memoryToItem 表实体 → 领域记忆项
func memoryToItem(m *entity.Memory) *memory.MemoryItem {
	item := &memory.MemoryItem{
		ID:      strconv.FormatUint(m.Id, 10),
		UserID:  m.UserId,
		Type:    m.Type,
		Content: m.Content,
	}
	if m.Tags != "" {
		if tags, err := jsonx.ToByStr[[]string](m.Tags); err == nil && tags != nil {
			item.Tags = *tags
		}
	}
	if m.Extra != nil {
		metadata := make(map[string]string, len(m.Extra))
		for k, v := range m.Extra {
			if s, ok := v.(string); ok {
				metadata[k] = s
			} else {
				metadata[k] = jsonx.ToStr(v)
			}
		}
		item.Metadata = metadata
	}
	if m.CreateTime != nil {
		item.CreatedAt = *m.CreateTime
	}
	if m.UpdateTime != nil {
		item.UpdatedAt = *m.UpdateTime
	}
	return item
}
