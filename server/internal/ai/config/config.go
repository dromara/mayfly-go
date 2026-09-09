package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"
	"strings"

	"github.com/spf13/cast"
)

const (
	ConfigKeyModel string = "AiModelConfig"

	// ConfigKeyAgent Agent 运行时配置（扩展裁剪 / 存储路径等）
	ConfigKeyAgent string = "AiAgentConfig"

	// DefaultMaxTokens 默认不限制输出 token（0 = 请求不带 max_tokens，由服务端
	// 使用模型默认输出上限）。thinking 模型的 reasoning 计入
	// max_tokens 预算，传小会导致回复被 finish_reason=length 截断；
	// 仅需控制成本时才显式配置 maxTokens
	DefaultMaxTokens = 0

	// DefaultContextWindow 默认模型上下文窗口（token）
	DefaultContextWindow = 128000

	// 默认存储目录（本地 JSONL 存储）
	DefaultSessionDir = "./sessions"
	DefaultMemoryDir  = "./memories"
)

type ModelConfig struct {
	Name          string  `json:"name"`  // 模型名称，主要用于展示
	Model         string  `json:"model"` // 模型标识，使用 协议/模型名 格式，如 openai/gpt-5.2
	BaseUrl       string  `json:"baseUrl"`
	ApiKey        string  `json:"apiKey"`  // api key
	TimeOut       int     `json:"timeOut"` // 请求超时时间，单位秒
	Temperature   float32 `json:"temperature"`
	MaxTokens     int     `json:"maxTokens"`
	ContextWindow int     `json:"contextWindow"` // 模型上下文窗口大小（token），用于压缩决策与 token 预算
	// EnableThinking 思考模式开关（仅 qwen3 系列等支持该参数的模型生效）：
	// nil = 未配置（qwen3 系列默认开启，对齐模型服务端默认行为）；
	// 思考模式下 qwen3 工具调用存在不稳定性——模型可能输出「正在执行 XX」
	// 的正文却不发 tool_calls，需关闭思考以提升工具调用稳定性
	EnableThinking *bool `json:"enableThinking"`
	// Retry 模型调用重试配置（eino v0.9 Model Retry 能力）；
	// nil = 不启用重试（默认，保持行为不变）。网络抖动 / 429 / 5xx 等
	// 瞬时失败自动重试，主动取消/超时不重试；退避为内置指数退避 + 抖动
	Retry *ModelRetryConfig `json:"retry"`
	// Failover 模型故障转移配置（eino v0.9 Model Failover 能力）；
	// nil = 不启用转移（默认，保持行为不变）。主模型经 Retry 重试耗尽后
	// 按序转移到 fallbacks 备用模型；主动取消/超时不转移
	Failover *ModelFailoverConfig `json:"failover"`
}

// ModelRetryConfig 模型调用重试配置
//
// 系统配置 AiModelConfig 中："retry": { "maxRetries": 2 }
//
// maxRetries 为最大重试次数（首次调用之外最多重试 N 次，0 视为不启用）
type ModelRetryConfig struct {
	MaxRetries int `json:"maxRetries"`
}

// ModelFailoverConfig 模型故障转移配置（eino v0.9 Model Failover 能力）
//
// 系统配置 AiModelConfig 中：
//
//	"failover": {
//	  "maxFailovers": 1,
//	  "fallbacks": [ { "model": "openai/gpt-4o-mini", "baseUrl": "...", "apiKey": "..." } ]
//	}
//
// fallbacks 按顺序作为第 1..N 次转移目标（复用 ModelConfig 结构，model 字段
// 须为 protocol/model 格式）；maxFailovers 限制转移次数（0 = 全部 fallbacks
// 可用）。备用模型实例同样经 protocol 缓存复用。组合语义：主模型先经 Retry
// 重试，重试耗尽后才触发转移
type ModelFailoverConfig struct {
	// MaxFailovers 最大转移次数（0 = 默认全部 fallbacks 可用）
	MaxFailovers int `json:"maxFailovers"`
	// Fallbacks 备用模型清单（按序转移）
	Fallbacks []*ModelConfig `json:"fallbacks"`
}

// parseModelFailoverConfig 解析 failover 配置段（容错：结构非法的条目跳过，
// 全部非法时返回 nil 不阻断装配）
func parseModelFailoverConfig(raw any) *ModelFailoverConfig {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	fc := &ModelFailoverConfig{MaxFailovers: cast.ToInt(m["maxFailovers"])}
	fallbacks, ok := m["fallbacks"].([]any)
	if !ok || len(fallbacks) == 0 {
		return nil
	}
	for _, rf := range fallbacks {
		fm, ok := rf.(map[string]any)
		if !ok {
			continue
		}
		if fb := parseFallbackModelConfig(fm); fb != nil {
			fc.Fallbacks = append(fc.Fallbacks, fb)
		}
	}
	if len(fc.Fallbacks) == 0 {
		return nil
	}
	return fc
}

// parseFallbackModelConfig 解析单个备用模型配置（model 字段必填，缺失视为非法条目）
func parseFallbackModelConfig(m map[string]any) *ModelConfig {
	conf := &ModelConfig{
		Name:    cast.ToString(m["name"]),
		Model:   cast.ToString(m["model"]),
		BaseUrl: cast.ToString(m["baseUrl"]),
		ApiKey:  cast.ToString(m["apiKey"]),
	}
	if conf.Model == "" {
		return nil
	}
	conf.TimeOut = cmp.Or(cast.ToInt(m["timeOut"]), 60)
	conf.Temperature = cmp.Or(cast.ToFloat32(m["temperature"]), 0.7)
	conf.MaxTokens = cmp.Or(cast.ToInt(m["maxTokens"]), DefaultMaxTokens)
	return conf
}

func (c *ModelConfig) GetModelSpec() ModelSpec {
	return ParseModel(c.Model)
}

// ModelSpec 定义模型规范
type ModelSpec struct {
	Protocol string
	Model    string
}

// ParseModel 解析模型字符串，格式为 "protocol/model"，如 "openai/gpt-3.5-turbo"
func ParseModel(model string) ModelSpec {
	parts := strings.SplitN(model, "/", 2)

	if len(parts) != 2 {
		return ModelSpec{
			Protocol: "openai", // 默认协议
			Model:    model,
		}
	}

	return ModelSpec{
		Protocol: parts[0],
		Model:    parts[1],
	}
}

func GetModel() *ModelConfig {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyModel)
	jm := c.GetJsonM()

	conf := new(ModelConfig)
	conf.Name = jm.GetStr("name")
	conf.Model = jm.GetStr("model")
	conf.BaseUrl = jm.GetStr("baseUrl")
	conf.ApiKey = jm.GetStr("apiKey")
	conf.TimeOut = cmp.Or(jm.GetInt("timeOut"), 60)
	conf.Temperature = cmp.Or(jm.GetFloat32("temperature"), 0.7)
	conf.MaxTokens = cmp.Or(jm.GetInt("maxTokens"), DefaultMaxTokens)
	conf.ContextWindow = cmp.Or(jm.GetInt("contextWindow"), DefaultContextWindow)
	if raw, exists := jm["enableThinking"]; exists {
		// 系统配置动态表单保存的值为字符串（"true"/"false"），空串视为未配置
		if s, isStr := raw.(string); !isStr || s != "" {
			b := cast.ToBool(raw)
			conf.EnableThinking = &b
		}
	} else if strings.Contains(strings.ToLower(conf.Model), "qwen") {
		// qwen3 系列默认开启思考（对齐模型服务端流式默认行为）
		t := true
		conf.EnableThinking = &t
	}
	if raw, exists := jm["retry"]; exists {
		if m, ok := raw.(map[string]any); ok {
			if retries := cast.ToInt(m["maxRetries"]); retries > 0 {
				conf.Retry = &ModelRetryConfig{MaxRetries: retries}
			}
		}
	}
	if raw, exists := jm["failover"]; exists {
		conf.Failover = parseModelFailoverConfig(raw)
	}
	return conf
}

// AgentConfig Agent 运行时配置
//
// 装配期生效：禁用清单在贡献者注册中心 Build 后经 WithFilter 裁剪（含
// 装配期激活型扩展：被裁剪的扩展不激活），存储目录决定本地 JSONL 存储
// 位置；修改后需重启服务。
//
// 多实例部署（企业级）：会话/记忆存储与扩展状态存储须为共享后端 ——
// 配置 Redis 后扩展状态自动切换为分布式缓存后端；会话/记忆的本地 JSONL
// 存储可通过 session.DefaultSessionStore 等注入点替换为共享实现。
type AgentConfig struct {
	// DisabledExtensions 禁用的贡献者 Id 列表（如 ["db_tools", "memory"]），
	// 可用 Id 见启动日志 contributors 清单（含 interrupt_approval /
	// interrupt_param_completion / memory_extraction 等装配期激活型扩展）
	DisabledExtensions []string `json:"disabledExtensions"`
	// SessionDir 会话历史存储目录（本地 JSONL 降级存储；application 层已装配
	// DB 后端（conversation + turn_item 表），本目录仅用于 DB 未装配时的单实例/单测场景）
	SessionDir string `json:"sessionDir"`
	// MemoryDir 长期记忆存储目录（本地 JSONL 降级存储；application 层已装配
	// DB 后端（t_ai_memory 表），本目录仅用于 DB 未装配时的单实例/单测场景）
	MemoryDir string `json:"memoryDir"`
	// ToolSearchThreshold 工具搜索阈值（eino v0.9 tool search middleware）：
	// 工具总量超过该值时，MCP 工具（mcp_ 前缀）转为 deferred —— 模型经
	// tool_search 元工具按需发现加载，避免大工具清单挤占上下文与 KV-cache。
	// 0 = 禁用（默认，全部工具直注，行为不变）。
	// 装配期生效：经 ResetDefaultAgent 重建 Agent 或重启服务后生效
	ToolSearchThreshold int `json:"toolSearchThreshold"`
}

// DisabledExtensionSet 禁用清单集合（WithFilter 参数形态）
func (c *AgentConfig) DisabledExtensionSet() map[string]struct{} {
	if len(c.DisabledExtensions) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(c.DisabledExtensions))
	for _, id := range c.DisabledExtensions {
		set[id] = struct{}{}
	}
	return set
}

func GetAgentConfig() *AgentConfig {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyAgent)
	jm := c.GetJsonM()

	conf := new(AgentConfig)
	if err := jm.Unmarshal("disabledExtensions", &conf.DisabledExtensions); err != nil {
		conf.DisabledExtensions = jm.GetStrSlice("disabledExtensions")
	}
	conf.SessionDir = cmp.Or(jm.GetStr("sessionDir"), DefaultSessionDir)
	conf.MemoryDir = cmp.Or(jm.GetStr("memoryDir"), DefaultMemoryDir)
	conf.ToolSearchThreshold = jm.GetInt("toolSearchThreshold")
	return conf
}
