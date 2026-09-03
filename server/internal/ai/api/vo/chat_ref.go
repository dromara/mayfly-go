package vo

// ChatSkillVO 聊天技能引用项（`/` 触发选择）
type ChatSkillVO struct {
	// Id 技能标识（即 skill code，供 skill_read 工具与 $mention 使用）
	Id string `json:"id"`
	// Name 技能名称
	Name string `json:"name"`
	// Description 技能描述
	Description string `json:"description"`
}

// ChatResourceVO 聊天资源引用项（`@` 触发选择，随消息下发使模型明确目标资源）
type ChatResourceVO struct {
	// Id 资源 id（字符串统一承载）
	Id string `json:"id"`
	// ResourceType 资源类型：machine / db
	ResourceType string `json:"resourceType"`
	// Name 资源名称
	Name string `json:"name"`
	// Code 资源编码（工具调用定位参数）
	Code string `json:"code"`
	// Ip 机器 IP（仅 machine）
	Ip string `json:"ip,omitempty"`
	// Port 机器端口（仅 machine）
	Port int `json:"port,omitempty"`
	// Description 辅助描述（机器为 ip:port，数据库为编码）
	Description string `json:"description"`
}
