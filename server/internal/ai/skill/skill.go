// Package skill 技能手册注册表（渐进式披露架构，对齐 tokhub plugin_skill）
//
// L1 索引层（系统提示词 preamble）：code + name + 描述（预算控制，由 agent 包
// skillCatalogContributor 渲染注入）；L3 内容层（skill_read 工具）：完整手册内容。
//
// 技能统一走 DB 管理（对齐菜单内置进 SQL 的方式：内置技能随迁移落库 t_ai_skill，
// 用户可在插件管理中编辑/删除），本包仅作为运行时只读视图：
// application.Init 注入 DB provider，List/Get 经 provider 读 DB，
// 读取失败时退回上次成功读取的缓存（进程内兜底，启动初期缓存为空）。
package skill

import (
	"sort"
	"strings"
	"sync"

	"mayfly-go/pkg/logx"
)

// Skill 技能手册
//
// 本结构为运行时视角（L1 目录 + L3 全文），DB 技能经 application 层转换后注入。
type Skill struct {
	// Code 技能标识，供 $code 显式提及与 skill_read 工具使用
	Code string
	// Name 技能名称（frontmatter name）
	Name string
	// Description 技能描述（frontmatter description，L1 目录展示）
	Description string
	// Content 完整内容（L3 层，skill_read 工具返回）
	Content string
}

// Registry 技能注册表（进程级单例，纯 DB 驱动）
//
// SetProvider 注入 DB 数据源后，List/Get 实时读取 DB（对齐 tokhub 逐请求
// 读取语义），读取失败时退回上次成功读取的缓存。
type Registry struct {
	mu sync.RWMutex
	// skills 上次成功读取的缓存（provider 失败时兜底）
	skills map[string]*Skill
	// provider 外部数据源（DB），由 application.Init 注入
	provider func() ([]*Skill, error)
}

// DefaultRegistry 默认技能注册表（包级单例，内容全部来自 DB provider）
var DefaultRegistry = NewRegistry()

// NewRegistry 创建空注册表
func NewRegistry() *Registry {
	return &Registry{skills: make(map[string]*Skill)}
}

// SetProvider 设置技能数据源（DB 驱动，application.Init 注入）
//
// 时序契约：须在技能目录首次被消费前调用（装配期 skill_injection /
// ChatSkills 接口 / extractExplicitSkillCodes 均会读取）；
// ListProviderSkills 读取失败时退回上次成功缓存，provider 短暂缺失不致中断
func (r *Registry) SetProvider(provider func() ([]*Skill, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.provider = provider
}

// ListProviderSkills 从数据源获取技能列表：provider 未设置或读取失败时
// 返回 nil（调用方退回上次缓存），读取结果同步刷新缓存（下次兜底用最新值）
func (r *Registry) ListProviderSkills() []*Skill {
	r.mu.RLock()
	provider := r.provider
	r.mu.RUnlock()
	if provider == nil {
		return nil
	}
	list, err := provider()
	if err != nil {
		logx.Warnf("[skill] load skills from provider failed, fallback to cache: %v", err)
		return nil
	}
	cache := make(map[string]*Skill, len(list))
	for _, s := range list {
		cache[s.Code] = s
	}
	r.mu.Lock()
	r.skills = cache
	r.mu.Unlock()
	return list
}

// Get 按 code 获取技能（provider 优先，失败退回缓存）
func (r *Registry) Get(code string) (*Skill, bool) {
	if list := r.ListProviderSkills(); list != nil {
		for _, s := range list {
			if s.Code == code {
				return s, true
			}
		}
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[code]
	return s, ok
}

// Exists 判断技能是否存在
func (r *Registry) Exists(code string) bool {
	_, ok := r.Get(code)
	return ok
}

// List 返回全部技能（按 Code 排序，保证目录渲染顺序稳定；provider 优先，失败退回缓存）
func (r *Registry) List() []*Skill {
	if list := r.ListProviderSkills(); list != nil {
		sorted := make([]*Skill, len(list))
		copy(sorted, list)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Code < sorted[j].Code })
		return sorted
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Skill, 0, len(r.skills))
	for _, s := range r.skills {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Code < list[j].Code })
	return list
}

// ParseFrontmatter 解析手册 frontmatter（--- 包裹的 name/description/allowed-tools）
func ParseFrontmatter(content string) (name, description, body string) {
	const fence = "---"
	if !strings.HasPrefix(content, fence) {
		return "", "", content
	}
	parts := strings.SplitN(content, fence, 3)
	if len(parts) < 3 {
		return "", "", content
	}
	for _, line := range strings.Split(parts[1], "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "name:"); ok {
			name = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(line, "description:"); ok {
			description = strings.TrimSpace(v)
		}
	}
	body = strings.TrimSpace(parts[2])
	return name, description, body
}

// BuildSkillMd 由元数据与正文构建 SKILL.md 内容（frontmatter + 正文）
//
// 对齐 tokhub build_skill_md：导出与内置技能落库时重建 frontmatter，
// 保证 zip 导入导出 roundtrip 后文件结构一致。
func BuildSkillMd(name, description, allowedTools, body string) string {
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("name: " + name + "\n")
	if description != "" {
		builder.WriteString("description: " + description + "\n")
	}
	if allowedTools != "" {
		builder.WriteString("allowed-tools: " + allowedTools + "\n")
	}
	builder.WriteString("---\n\n")
	builder.WriteString(body)
	return builder.String()
}
