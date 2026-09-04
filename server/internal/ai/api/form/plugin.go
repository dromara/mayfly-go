package form

import "encoding/json"

// SkillSaveRequest 技能创建/更新请求（instructions 为 SKILL.md 正文）
type SkillSaveRequest struct {
	Code         string `json:"code"`
	Description  string `json:"description"`
	AllowedTools string `json:"allowedTools"`
	Instructions string `json:"instructions"`
}

// SkillResourceRequest 技能资源新增/更新请求（body 含 path + content 即 upsert）
type SkillResourceRequest struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content"`
}

// PluginInstanceSaveRequest 插件实例创建/更新请求（config 为类型化配置对象，
// 后端转 JSON 字符串落库，schema 由 PluginTypeHandler 约定）
type PluginInstanceSaveRequest struct {
	PluginType  string          `json:"pluginType" binding:"required"`
	Code        string          `json:"code" binding:"required"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"` // 类型化配置对象（如 mcp: {url, headers, timeoutSec}）
}

// PluginInstanceToggleRequest 插件实例启停请求
// （独立请求体：不强制携带 name 等更新字段）
type PluginInstanceToggleRequest struct {
	Enabled int `json:"enabled"` // 1=启用 0=停用
}
