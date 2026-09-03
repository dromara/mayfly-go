package memoryext

import (
	"context"
	"fmt"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/memory"
	"mayfly-go/pkg/contextx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// maxSearchLimit 单次搜索返回上限（防止撑爆上下文）
const maxSearchLimit = 20

// defaultSearchLimit 未指定条数时的默认返回数
const defaultSearchLimit = 5

// MemorySearchParam memory_search 工具入参
type MemorySearchParam struct {
	// Query 搜索关键词（空格分隔多个词，任一命中即返回；留空返回最近记忆）
	Query string `json:"query" jsonschema_description:"搜索关键词，空格分隔多个词，任一命中即返回；留空则返回最近的记忆"`
	// Limit 返回条数上限（默认 5，最大 20）
	Limit int `json:"limit" jsonschema_description:"返回条数上限，默认 5，最大 20"`
}

// MemorySearchItem 搜索结果记忆项
type MemorySearchItem struct {
	// Id 记忆 ID
	Id string `json:"id"`
	// Type 记忆类型：preference/fact/skill/experience
	Type string `json:"type"`
	// Content 记忆内容
	Content string `json:"content"`
	// Tags 标签
	Tags []string `json:"tags,omitempty"`
}

// MemorySearchOutput memory_search 工具出参
type MemorySearchOutput struct {
	// Items 命中的记忆列表（按近期优先排序）
	Items []MemorySearchItem `json:"items"`
}

// MemoryToolExtension 记忆搜索工具扩展（ToolContributor 通道）
//
// 贡献 memory_search 工具：LLM 可主动检索当前用户的长期记忆，
// 与 memory 片段注入（ContributeTurnContext）互补——注入的是近期记忆
// 概览，工具用于按需精确召回。业务插件注册同名工具即可覆盖
// （后注册胜出），或经配置按 Id（memory_tools）裁剪。
type MemoryToolExtension struct {
	memoryManager *memory.Manager
}

// NewToolExtension 创建记忆搜索工具扩展（memoryManager 为 nil 时 Tools 返回空，无工具贡献）
func NewToolExtension(memoryManager *memory.Manager) *MemoryToolExtension {
	return &MemoryToolExtension{memoryManager: memoryManager}
}

var _ contributor.ToolContributor = (*MemoryToolExtension)(nil)

func (e *MemoryToolExtension) Id() string { return "memory_tools" }

func (e *MemoryToolExtension) Tools(_ context.Context, _ *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	if e.memoryManager == nil {
		return nil, nil
	}
	t, err := utils.InferTool("memory_search",
		"搜索当前用户的历史记忆（偏好、事实、经验等）。当需要回顾用户此前告知的信息或既往操作习惯时调用；也可用于确认某条记忆是否存在。",
		func(ctx context.Context, param *MemorySearchParam) (*MemorySearchOutput, error) {
			// 默认 Agent 为进程级单例，工具贡献期无用户维度，
			// 运行期经请求上下文解析当前登录用户（多实例下各实例一致）
			la := contextx.GetLoginAccount(ctx)
			if la == nil {
				return nil, fmt.Errorf("memory_search requires a login user context")
			}
			limit := param.Limit
			if limit <= 0 {
				limit = defaultSearchLimit
			}
			if limit > maxSearchLimit {
				limit = maxSearchLimit
			}
			userID := fmt.Sprintf("%d", la.Id)
			items, err := e.memoryManager.Search(ctx, userID, param.Query, limit)
			if err != nil {
				return nil, err
			}
			result := make([]MemorySearchItem, 0, len(items))
			for _, item := range items {
				result = append(result, MemorySearchItem{
					Id:      item.ID,
					Type:    item.Type,
					Content: item.Content,
					Tags:    item.Tags,
				})
			}
			return &MemorySearchOutput{Items: result}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{t}, nil
}

// InstallTools 注册记忆工具扩展到 Builder
func InstallTools(b *contributor.Builder, memoryManager *memory.Manager) {
	b.RegisterTool(NewToolExtension(memoryManager))
}
