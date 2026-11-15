// Package textgen 定义「当前生效模型」的一次性文本生成端口。
//
// 存在动机（依赖倒置，破除循环依赖）：业务模块（如 db 的 SQL 草稿生成）需要调用大模型，
// 但业务 application 直接 import ai/application 会成环——ai/application/resource 已反向
// 依赖 db/application。故把「生成一段文本」下沉为中立契约包：ai/application 在 Init 期
// 注册实现，业务侧面向端口编程，二者之间不留直接依赖。与 ai/pkg/headless 同一手法。
//
// 本包只定义「一次调用、无工具、无会话」的补全语义：
//   - 多步编排/工具调用属于 Agent，不在这里；
//   - 端口不做业务分派：Purpose 只用于日志与计量打标，模型档位、提示词、校验全部在消费方；
//   - 实现方每次调用都必须经 llm.Provider 取当前生效模型，不得在装配期固化模型实例，
//     否则管理员改了地址/Key 之后本端口仍走旧端点（与摘要器同一契约）。
//
// 本包是叶子包：只依赖标准库与 eino schema，供任意业务模块经此取用。
package textgen

import (
	"context"
	"sync"
	"time"
	"unicode/utf8"
)

// Request 一次文本生成请求
type Request struct {
	// Purpose 用途打标（如 sql_draft），仅用于日志/计量，端口不据此分支
	Purpose string
	// System 系统提示词（角色与硬性约束）
	System string
	// User 用户内容（待处理材料 + 任务）
	User string
	// MaxTokens 输出上限（<=0 表示不传，交给服务端默认；thinking 模型的 reasoning 计入该预算）
	MaxTokens int
	// Temperature 采样温度（草稿类固定 0；nil 表示不传）
	Temperature *float32
	// Timeout 单次调用超时（<=0 表示只受外层 ctx 约束）
	Timeout time.Duration
	// PreferFast 请求「快速通道」：短结构化输出（SQL 草稿、选表、行内补全）用思考模式
	// 只增加延迟不提升质量。实现方仅在当前配置**显式开启**思考时才降级为关闭，
	// 未开启（或网关不支持该参数）时按原配置执行，行为与默认一致
	PreferFast bool
}

// Usage 本轮 token 用量（模型未上报时各字段为 0）
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// Port 一次性文本生成端口
type Port interface {
	// Complete 生成一段文本。返回 (文本, 用量, 错误)，用量可为 nil
	Complete(ctx context.Context, req *Request) (string, *Usage, error)
	// Available 当前是否可用（模型三要素是否齐备）。只回布尔与错误，不回任何配置原文
	Available(ctx context.Context) (bool, error)
}

var (
	portMu sync.RWMutex
	port   Port
)

// Register 注册端口实现（ai/application.Init 调用；后注册覆盖先注册）
func Register(p Port) {
	portMu.Lock()
	defer portMu.Unlock()
	port = p
}

// Get 取端口实现；未注册返回 false，调用方须自行降级（不得当成「无限制」继续）
func Get() (Port, bool) {
	portMu.RLock()
	defer portMu.RUnlock()
	return port, port != nil
}

// IsAvailable 便捷判定：实现未注册或模型未配置均返回 false（错误一并回传供日志）
func IsAvailable(ctx context.Context) (bool, error) {
	p, ok := Get()
	if !ok {
		return false, nil
	}
	return p.Available(ctx)
}

// EstimateTokens 粗略估算 token 数（1 token ≈ 4 字符）。
//
// 与 ai/agent/contributor 的口径保持一致：预算裁剪只需要「同一量纲的近似值」，
// 引入真实 tokenizer 会让业务侧多一个重依赖，却没改变「按预算丢片段」的决策质量。
func EstimateTokens(text string) int {
	return (utf8.RuneCountInString(text) + 3) / 4
}
