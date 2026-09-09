package application

import (
	"encoding/json"
	"strings"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/pkg/errorx"
)

// PluginTypeHandler 插件类型处理器（CapabilityHandler，代码级类型注册表）
//
// 不建 t_plugin_definition 表：mayfly-go 无多租户/市场场景，插件类型固定由代码
// 注册声明；新增插件类型 = 实现本接口 + Init 中注册一行，核心流程零改动（开闭原则）。
type PluginTypeHandler interface {
	// TypeCode 类型标识（写入 t_ai_plugin_instance.plugin_type）
	TypeCode() string
	// ValidateConfig 校验实例 config（JSON 字符串）是否符合该类型 schema 约定
	ValidateConfig(config string) error
}

// PluginTypeInfo 类型元信息（GET /plugin/types 下发，前端类型选择器数据源）
type PluginTypeInfo struct {
	Code string `json:"code"`
}

// pluginTypeRegistry 类型注册表（Init 装配后只读，无运行期写入）
var pluginTypeRegistry []PluginTypeHandler

// RegisterPluginType 注册插件类型（重复注册同类型报错，暴露装配期错误）
func RegisterPluginType(h PluginTypeHandler) {
	for _, exist := range pluginTypeRegistry {
		if exist.TypeCode() == h.TypeCode() {
			panic("plugin type already registered: " + h.TypeCode())
		}
	}
	pluginTypeRegistry = append(pluginTypeRegistry, h)
}

// GetPluginType 按类型标识取处理器
func GetPluginType(pluginType string) (PluginTypeHandler, bool) {
	for _, h := range pluginTypeRegistry {
		if h.TypeCode() == pluginType {
			return h, true
		}
	}
	return nil, false
}

// ListPluginTypes 全部已注册类型（注册序）
func ListPluginTypes() []PluginTypeInfo {
	types := make([]PluginTypeInfo, 0, len(pluginTypeRegistry))
	for _, h := range pluginTypeRegistry {
		types = append(types, PluginTypeInfo{Code: h.TypeCode()})
	}
	return types
}

// ============== 类型化 config 结构（各类型 schema 的 Go 契约） ==============

// SkillInstanceConfig 技能插件 config（引用 t_ai_skill，Managed 模式对应 SkillSource::Managed）
type SkillInstanceConfig struct {
	// SkillCode 引用的技能 code（t_ai_skill.code）
	SkillCode string `json:"skillCode"`
}

// McpInstanceConfig MCP 插件 config（连接配置内联，config_template + auth_ref 简化版）
type McpInstanceConfig struct {
	// Url MCP 服务器地址（Streamable HTTP / SSE）
	Url string `json:"url"`
	// Headers 请求头 JSON 字符串，如 {"Authorization":"Bearer xx"}
	Headers string `json:"headers"`
	// TimeoutSec 请求超时秒数（<=0 时取默认 30）
	TimeoutSec int `json:"timeoutSec"`
}

// parseInstanceConfig 解析 config JSON 到目标结构（解析失败返回错误，由调用方决定容错策略）
func parseInstanceConfig[T any](config string) (*T, error) {
	var parsed T
	if err := json.Unmarshal([]byte(config), &parsed); err != nil {
		return nil, errorx.NewBizf("invalid instance config: %v", err)
	}
	return &parsed, nil
}

// mustSkillConfig 解析技能实例 config（空/非法返回零值，运行期容错不阻断装配）
func mustSkillConfig(config string) *SkillInstanceConfig {
	parsed, err := parseInstanceConfig[SkillInstanceConfig](config)
	if err != nil || parsed == nil {
		return &SkillInstanceConfig{}
	}
	return parsed
}

// mustMcpConfig 解析 MCP 实例 config（超时缺省 30s，运行期容错不阻断装配）
func mustMcpConfig(config string) *McpInstanceConfig {
	parsed, err := parseInstanceConfig[McpInstanceConfig](config)
	if err != nil || parsed == nil {
		return &McpInstanceConfig{TimeoutSec: 30}
	}
	if parsed.TimeoutSec <= 0 {
		parsed.TimeoutSec = 30
	}
	return parsed
}

// ============== 内置类型实现 ==============

// skillInstanceType 技能插件类型
type skillInstanceType struct{}

func (skillInstanceType) TypeCode() string { return entity.PluginTypeSkill }

func (skillInstanceType) ValidateConfig(config string) error {
	parsed, err := parseInstanceConfig[SkillInstanceConfig](config)
	if err != nil {
		return err
	}
	if strings.TrimSpace(parsed.SkillCode) == "" {
		return errorx.NewBiz("skill instance config requires non-empty skillCode")
	}
	return nil
}

// mcpInstanceType MCP 插件类型
type mcpInstanceType struct{}

func (mcpInstanceType) TypeCode() string { return entity.PluginTypeMcp }

func (mcpInstanceType) ValidateConfig(config string) error {
	parsed, err := parseInstanceConfig[McpInstanceConfig](config)
	if err != nil {
		return err
	}
	url := strings.TrimSpace(parsed.Url)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return errorx.NewBiz("mcp instance config requires http(s) url")
	}
	// headers 可空，非空须为合法 JSON 对象
	if trimmed := strings.TrimSpace(parsed.Headers); trimmed != "" {
		headers := map[string]string{}
		if err := json.Unmarshal([]byte(trimmed), &headers); err != nil {
			return errorx.NewBiz("mcp instance headers must be a valid JSON object")
		}
	}
	return nil
}
