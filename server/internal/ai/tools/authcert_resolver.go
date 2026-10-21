package tools

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
)

// AuthCertResolver 把「机器标识」（机器名 / ip:port / 裸 ip / 机器 code）解析为该机器**唯一**
// 可操作凭证名；有多张凭证或无法确定时必须返回 false（不猜）。
type AuthCertResolver func(ctx context.Context, input string) (string, bool)

var (
	authCertMu sync.RWMutex
	// 默认是「什么都解析不出来」的空实现：未注册时所有调用方走"不猜"分支，
	// 比 nil 检查散落在各调用点更可靠（漏一处就是 panic）
	authCertResolver AuthCertResolver = func(context.Context, string) (string, bool) { return "", false }
)

// DeclareAuthCertResolver 注册机器凭证解析器（机器工具包在 init 里调用）。
//
// 为什么用注册而不是直接调用：解析需要账号可见的资源清单（ai/application/resource 服务），而本
// resolver 的消费点 NormalizeActionParams 位于 tools **父包**、由提案登记期调用；父包不能 import
// application（application 反向 import tools 来装配具体工具，父包再 import application 即编译期成环）。
// 故由具体工具**子包**把实现注册进来——machinetool 自身 import application/resource 属正向依赖，不成环。
// 与 DeclareCapability / DeclareParamSchema 同一模式：工具侧在注册处声明能力，下层父包只存结果、不猜。
//
// 注册必须在包初始化期完成，不能放在工具构造函数体内——消费点（提案登记时的参数规范化）
// 可能早于任何一次工具构造，届时表为空会静默不规范化（同类时机缺陷端到端实测过）
func DeclareAuthCertResolver(resolver AuthCertResolver) {
	if resolver == nil {
		return
	}
	authCertMu.Lock()
	defer authCertMu.Unlock()
	authCertResolver = resolver
}

// ResolveAuthCertName 解析机器标识为真实凭证名。
//
// 返回 false 的三种情形都刻意不猜：未注册解析器、该机器有多张凭证、输入已是要么合法凭证名要么
// 匹配不到机器
func ResolveAuthCertName(ctx context.Context, input string) (string, bool) {
	authCertMu.RLock()
	resolver := authCertResolver
	authCertMu.RUnlock()
	return resolver(ctx, input)
}

// machineActionName 机器命令执行动作名：只有它需要规范化 authCertName。
//
// 为什么不按「只要有 authCertName 键就解」：数据库动作也有 authCertName（实例凭证），
// 而本解析器只查机器资源，一个实例凭证名恰好等于某台机器名时会被错改。宁可窄，不可误改
const machineActionName = ToolNameMachineCommandExec

// NormalizeActionParams 把机器命令提案参数里的凭证标识替换为该机器唯一可操作凭证名。
//
// 存在动机（端到端实测到）：模型会把机器名写进 authCertName（如 "aliyun"）——键合法、执行时
// 也会被兜底纠正，但提案文本是给审批人看的，且放行要求「批准参数 == 执行实收参数」逐项相等，
// 不规范化会让审批记录的凭证与实际连接的凭证字面不一致，审计上无法对应。
//
// 保守分岔一律原样返回：非机器动作、脏 JSON、无该键、解析不出（多凭证就是不猜）、
// 解出来与原值相同——宁可不规范化，也不能凭空改坏审批人已据以判断的参数
func NormalizeActionParams(ctx context.Context, action, params string) string {
	if strings.TrimSpace(action) != machineActionName {
		return params
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(params)), &parsed); err != nil || parsed == nil {
		return params
	}
	raw, ok := parsed[machineActionCertParam].(string)
	if !ok {
		return params
	}
	cert, resolved := ResolveAuthCertName(ctx, raw)
	if !resolved || cert == raw {
		return params
	}
	parsed[machineActionCertParam] = cert
	out, err := json.Marshal(parsed)
	if err != nil {
		return params
	}
	return string(out)
}

// machineActionCertParam 机器动作里承载凭证的参数名（与 CommandExecParam 的 json tag 一致）
const machineActionCertParam = "authCertName"
