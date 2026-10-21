package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mayfly-go/pkg/utils/jsonx"
)

// RunGrant 一次性运行许可：本次运行只允许调用清单内的工具，其余一律拒绝。
//
// 存在动机（capability 策略表达不了的那一半约束）：策略是「档」级的静态声明，能说
// 「诊断档只许调用只读工具」，却说不了「这一次运行只许执行经人工审批锁定的那一个动作」。
// 受控执行档缺了这一维，「参数锁定」就只剩提示词约束——而提示词不是机制：模型在工具报错、
// 上下文被截断或换模型时换一个高危工具、改一组参数，恰恰是本特性要防的场景。
//
// 语义（两个维度都是天花板，取交集）：
//   - 未下发许可（nil）：不因本次运行额外收窄，能力策略说了算
//   - 下发了许可：清单外的工具名一律拒绝（即使该档策略本来放行它的 capability）
//   - 档位声明「必须由许可驱动」（见 AgentProfile.RequireRunGrant）时，没有许可就什么都不许做
//
// 许可随运行上下文下发，不进 Agent 装配：eino 的工具清单构造后不可刷新，而每次运行的许可都不同，
// 若把它做成装配参数会退化成「每次执行都重建一个 Agent」。
type RunGrant struct {
	// AllowedToolNames 本次运行允许调用的工具名
	AllowedToolNames []string
	// ApprovedTools 已由人工批准的工具名 → 批准时锁定的参数 JSON。
	//
	// 存在动机：受控执行那一轮里工具自身还会要求人工审批（见 InterruptOrResumeApproval），
	// 而无头运行没有审批应答者——于是「人已经批准的变更」一次也执行不了（实测：run 不报错、
	// 被记成 succeeded，提案被记成 executed，而实际什么都没做）。
	//
	// 为什么必须连参数一起绑：只按工具名放行等于「批准过一次，之后该工具传什么参数都免审批」，
	// 模型在执行那轮里换一组参数（批准的是 command:"true"，跑成 DROP TABLE）就能绕过，
	// 「参数锁定」就退回提示词约束
	ApprovedTools map[string]string
}

// ApprovedFor 判定「该工具 + 该组参数」是否已被人工批准。
//
// 参数逐项比对（两边都先展平成 map[string]string），规避 JSON 键序与数字/字符串表示差异；
// 解析失败、无批准内容、批准项取值不一致、执行时夹带实质参数都判未批准
// ——宁可回到审批（无头运行则诚实失败），也不放行一个可疑调用
func (g *RunGrant) ApprovedFor(toolName string, arguments any) bool {
	if g == nil {
		return false
	}
	approved, ok := g.ApprovedTools[toolName]
	if !ok {
		return false
	}
	want, ok1 := flattenParams(approved)
	got, ok2 := flattenParams(jsonx.ToStr(arguments))
	if !ok1 || !ok2 {
		return false
	}

	// 批准的每一项都必须逐字命中：批准里给了值、执行时没给或给了不同值，都是改动批准内容
	for key, value := range want {
		gotValue, exists := got[key]
		if !exists || gotValue != value {
			return false
		}
	}

	// 执行侧多出的键只允许是零值形态：工具实收参数由参数 struct marshal 得出，必然带上
	// 批准时未提及的可选字段（如 dbId:0、dbName:""），那等价于「没指定」，不构成改动。
	//
	// 为什么不用「两边键数相等」当判据：那会要求提案必须写全每一个参数才能对齐，
	// 而形状校验只强证必填项——省略了可选参数的提案能登记、能被批准，却永远不放行（真机核实）
	for key, value := range got {
		if _, approvedKey := want[key]; approvedKey {
			continue
		}
		if !isZeroLikeParam(value) {
			return false
		}
	}
	return true
}

// isZeroLikeParam 参数值的「未指定」形态（展平后的字符串表示）。
//
// 保守集合：只把结构体零值对应的写法当成「没指定」；任何实质值都不算
func isZeroLikeParam(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "0", "false", "null", "<nil>":
		return true
	}
	return false
}

// flattenParams 把参数 JSON 展平成 map[string]string（值统一用 %v 表示，使 1 与 "1" 可比）。
// 空参数返回 false：没有可比对的内容就没有凭据，一律不放行
func flattenParams(raw string) (map[string]string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(trimmed), &m); err != nil {
		return nil, false
	}
	out := make(map[string]string, len(m))
	for key, value := range m {
		out[key] = fmt.Sprintf("%v", value)
	}
	return out, true
}

type runGrantKey struct{}

// WithRunGrant 把一次性许可注入运行上下文；grant 为 nil 时原样返回（不注入即不收窄）
func WithRunGrant(ctx context.Context, grant *RunGrant) context.Context {
	if grant == nil {
		return ctx
	}
	return context.WithValue(ctx, runGrantKey{}, grant)
}

// RunGrantFrom 读取运行上下文中的一次性许可（不存在返回 nil）
func RunGrantFrom(ctx context.Context) *RunGrant {
	if ctx == nil {
		return nil
	}
	grant, _ := ctx.Value(runGrantKey{}).(*RunGrant)
	return grant
}

// Allows 判定工具名是否在许可清单内
func (g *RunGrant) Allows(toolName string) bool {
	if g == nil {
		return false
	}
	for _, name := range g.AllowedToolNames {
		if name == toolName {
			return true
		}
	}
	return false
}
