package agent

import (
	"mayfly-go/internal/ai/prompt"
	"mayfly-go/pkg/logx"
)

// SystemInstruction 渲染静态系统提示词（注入 adk Instruction）
//
// 从 embed 模板 internal/system_prompt.md 读取并做模板解析；
// 渲染失败返回空串并记录告警（agent 降级为无系统提示词运行，不阻断启动）。
func SystemInstruction() string {
	sp, err := prompt.GetPrompt("internal/system_prompt.md", nil)
	if err != nil {
		logx.Warnf("load system prompt failed, agent will run without system prompt: %v", err)
		return ""
	}
	return sp
}
