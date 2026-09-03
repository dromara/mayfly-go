package form

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

// McpServerSaveRequest MCP 服务器创建/更新请求
type McpServerSaveRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Url         string `json:"url" binding:"required"`
	Headers     string `json:"headers"` // JSON，如 {"Authorization":"Bearer xx"}
	TimeoutSec  int    `json:"timeoutSec"`
	Enabled     int    `json:"enabled"` // 1=启用
}
