package memory

import (
	"context"
	"fmt"
	"mayfly-go/pkg/logx"
	"strings"
	"sync"
)

// Manager 统一的记忆管理器
type Manager struct {
	store     Store
	extractor Extractor // 记忆提取器
	config    *Config   // 记忆配置
	mu        sync.RWMutex
}

// Config 记忆配置
type Config struct {
	Enabled         bool    // 是否启用记忆功能
	DefaultTTL      int64   // 默认过期时间（秒），0表示不过期
	MaxItemsPerUser int     // 每个用户最大记忆数量
	MinConfidence   float64 // 最小置信度阈值
}

// DefaultConfig 返回默认记忆配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:         true,
		DefaultTTL:      3600, // 1小时
		MaxItemsPerUser: 100,
		MinConfidence:   0.5,
	}
}

// NewManager 创建记忆管理器
func NewManager(store Store) *Manager {
	return &Manager{
		store:     store,
		extractor: nil, // 不默认设置提取器，需要外部显式配置
		config:    DefaultConfig(),
	}
}

// WithExtractor 设置记忆提取器
func (m *Manager) WithExtractor(extractor Extractor) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()
	if extractor != nil {
		m.extractor = extractor
	}
	return m
}

// WithConfig 设置记忆配置
func (m *Manager) WithConfig(config *Config) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()
	if config != nil {
		m.config = config
	}
	return m
}

// ==================== 长期记忆操作 ====================

// Save 保存记忆
func (m *Manager) Save(ctx context.Context, item *MemoryItem) error {
	m.mu.RLock()
	enabled := m.config.Enabled
	m.mu.RUnlock()

	if !enabled {
		return nil
	}

	// 保存记忆
	if err := m.store.Save(ctx, []*MemoryItem{item}); err != nil {
		return fmt.Errorf("save memory: %w", err)
	}

	logx.InfofContext(ctx, "stored memory: type=%s, content=%s", item.Type, item.Content)
	return nil
}

// SaveBatch 批量保存记忆
func (m *Manager) SaveBatch(ctx context.Context, items []*MemoryItem) error {
	if len(items) == 0 {
		return nil
	}

	m.mu.RLock()
	enabled := m.config.Enabled
	m.mu.RUnlock()

	if !enabled {
		return nil
	}

	if err := m.store.Save(ctx, items); err != nil {
		return fmt.Errorf("batch save memories: %w", err)
	}

	logx.InfofContext(ctx, "stored %d memories", len(items))
	return nil
}

// RetrieveByTags 根据标签检索记忆
func (m *Manager) RetrieveByTags(ctx context.Context, userID string, tags []string) ([]*MemoryItem, error) {
	m.mu.RLock()
	enabled := m.config.Enabled
	m.mu.RUnlock()

	if !enabled {
		return []*MemoryItem{}, nil
	}

	items, err := m.store.GetByUser(ctx, userID, tags)
	if err != nil {
		return nil, fmt.Errorf("retrieve memories: %w", err)
	}

	logx.DebugfContext(ctx, "retrieved %d memories for user %s", len(items), userID)
	return items, nil
}

// RetrieveAll 检索用户的所有记忆
func (m *Manager) RetrieveAll(ctx context.Context, userID string) ([]*MemoryItem, error) {
	return m.RetrieveByTags(ctx, userID, nil)
}

// Search 语义搜索记忆
func (m *Manager) Search(ctx context.Context, userID string, query string, limit int) ([]*MemoryItem, error) {
	m.mu.RLock()
	enabled := m.config.Enabled
	m.mu.RUnlock()

	if !enabled {
		return []*MemoryItem{}, nil
	}

	items, err := m.store.Search(ctx, userID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}

	logx.DebugfContext(ctx, "searched %d memories for user %s with query: %s", len(items), userID, query)
	return items, nil
}

// Delete 删除指定的记忆
func (m *Manager) Delete(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	if err := m.store.Delete(ctx, userID, ids); err != nil {
		return fmt.Errorf("delete memories: %w", err)
	}

	logx.InfofContext(ctx, "deleted %d memories for user %s", len(ids), userID)
	return nil
}

// ExtractAndSave 从消息中提取记忆并自动保存
func (m *Manager) ExtractAndSave(ctx context.Context, req *ExtractMemoryReq) error {
	m.mu.RLock()
	extractor := m.extractor
	enabled := m.config.Enabled
	m.mu.RUnlock()

	if !enabled || extractor == nil {
		return nil
	}

	// 提取记忆
	memories, err := extractor.ExtractFromMessages(ctx, req.UserId, req.Msgs)
	if err != nil {
		return fmt.Errorf("extract memories: %w", err)
	}

	if len(memories) == 0 {
		return nil
	}

	// 批量保存记忆
	return m.SaveBatch(ctx, memories)
}

// CreateMemory 创建记忆项的辅助函数
func CreateMemory(userID string, memType string, content string, tags []string) *MemoryItem {
	return &MemoryItem{
		UserID:   userID,
		Type:     memType,
		Content:  content,
		Tags:     tags,
		Metadata: make(map[string]string),
	}
}

// BuildMemoryMessage 构建记忆注入片段文本
// 该方法封装了记忆检索和格式化的完整流程；无可用记忆时返回空串
func (m *Manager) BuildMemoryMessage(ctx context.Context, userID string) string {
	if m == nil || userID == "" {
		return ""
	}

	m.mu.RLock()
	enabled := m.config.Enabled
	m.mu.RUnlock()

	if !enabled {
		return ""
	}

	// 检索相关记忆（使用语义搜索）
	memories, err := m.Search(ctx, userID, "", 10) // 默认返回最近10条
	if err != nil {
		logx.WarnfContext(ctx, "retrieve memories error: %v", err)
		return ""
	}

	if len(memories) == 0 {
		return ""
	}

	// 格式化记忆为文本
	memoryText := m.formatMemories(memories)
	if memoryText == "" {
		return ""
	}

	logx.InfofContext(ctx, "injected %d memories into context", len(memories))
	return fmt.Sprintf("[用户记忆]\n%s\n\n[请根据以上记忆信息提供更个性化的服务]", memoryText)
}

// formatMemories 格式化记忆列表为文本
func (m *Manager) formatMemories(memories []*MemoryItem) string {
	if len(memories) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, mem := range memories {
		builder.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, mem.Type, mem.Content))
		if len(mem.Tags) > 0 {
			builder.WriteString(fmt.Sprintf("   标签: %s\n", strings.Join(mem.Tags, ", ")))
		}
	}
	return builder.String()
}
