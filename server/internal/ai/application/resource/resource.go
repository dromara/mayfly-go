// Package resource AI 聊天统一的用户可操作资源查询服务
//
// 聚合各业务模块（machine/db/...）的资源清单，统一做账号级权限过滤（基于
// 团队资源分配的标签授权），供资源清单查询工具（ListResources）与工具参数
// 补全选项构建共用同一数据源，避免各处自行查询导致口径不一致或越权泄露。
//
// 扩展新资源类型（开闭原则）：实现 Provider 并 RegisterProvider 注册即可，
// 核心查询链路与工具层零改动。
package resource

import (
	"context"
	"sort"
	"strings"
	"sync"

	"mayfly-go/pkg/utils/collx"
)

// 资源类型标识（与协议层/前端资源类型口径一致）
const (
	TypeMachine = "machine"
	TypeDb      = "db"
)

// Extra 通用键：类型内部复用字段（参数补全选项 payload 构建等场景取用）
const (
	ExtraKeyAuthCertName    = "authCertName"    // 授权凭证名（机器的 code 即凭证名）
	ExtraKeyIp              = "ip"              // 机器 IP
	ExtraKeyPort            = "port"            // 机器端口
	ExtraKeyGetDatabaseMode = "getDatabaseMode" // db：库名获取方式（dbentity.DbGetDatabaseMode）
	ExtraKeyDatabase        = "database"        // db：指定库名（空格分隔多个）
)

// Resource 统一资源描述
type Resource struct {
	// Type 资源类型（machine/db/...）
	Type string
	// Id 资源 id（字符串统一承载）
	Id string
	// Code 资源编码（工具调用定位参数）
	Code string
	// Name 资源名称
	Name string
	// Description 辅助描述（机器为 ip:port，数据库为编码）
	Description string
	// Detail 面向 LLM/展示的公开明细（机器为 ip/port，数据库为可连接库名 databases）
	Detail map[string]string
	// Extra 类型内部复用字段（参数补全 payload 构建等，不直接透出给 LLM）
	Extra collx.M
}

// Provider 资源提供者契约：每种资源类型一个实现，
// 返回指定账号有权限操作的资源清单（keyword 过滤由 App 层统一处理）
type Provider interface {
	// Type 资源类型标识
	Type() string
	// List 返回指定账号有权限操作的全部资源
	List(ctx context.Context, accountId uint64) ([]*Resource, error)
}

// Query 资源查询条件
type Query struct {
	// Types 类型过滤，空为全部（未注册的类型自动跳过）
	Types []string
	// Keyword 关键词，对 Name/Code/Description 大小写不敏感模糊匹配
	Keyword string
}

// App 统一资源查询门面
type App interface {
	// Types 已注册的资源类型（稳定排序）
	Types() []string
	// RegisterProvider 注册资源提供者，同类型后注册覆盖先注册（对齐项目覆盖约定）
	RegisterProvider(p Provider)
	// List 按条件查询账号有权限操作的资源；任一 provider 失败则整体失败（保证结果完整性）
	List(ctx context.Context, accountId uint64, query *Query) ([]*Resource, error)
}

type appImpl struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewApp 创建资源查询 App（零 provider，装配方按需注册）
func NewApp() App {
	return &appImpl{providers: make(map[string]Provider)}
}

func (a *appImpl) Types() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	types := make([]string, 0, len(a.providers))
	for t := range a.providers {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func (a *appImpl) RegisterProvider(p Provider) {
	if p == nil || p.Type() == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.providers[p.Type()] = p
}

func (a *appImpl) List(ctx context.Context, accountId uint64, query *Query) ([]*Resource, error) {
	if query == nil {
		query = &Query{}
	}
	wantTypes := query.Types
	if len(wantTypes) == 0 {
		wantTypes = a.Types()
	}

	result := make([]*Resource, 0)
	for _, t := range wantTypes {
		a.mu.RLock()
		p, ok := a.providers[t]
		a.mu.RUnlock()
		if !ok {
			continue
		}
		resources, err := p.List(ctx, accountId)
		if err != nil {
			return nil, err
		}
		result = append(result, resources...)
	}
	return matchKeyword(result, query.Keyword), nil
}

// matchKeyword 关键词过滤：Name/Code/Description 大小写不敏感包含匹配
func matchKeyword(resources []*Resource, keyword string) []*Resource {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return resources
	}
	filtered := make([]*Resource, 0, len(resources))
	for _, r := range resources {
		if r == nil {
			continue
		}
		if strings.Contains(strings.ToLower(r.Name), keyword) ||
			strings.Contains(strings.ToLower(r.Code), keyword) ||
			strings.Contains(strings.ToLower(r.Description), keyword) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
