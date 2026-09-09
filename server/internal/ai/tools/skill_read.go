package tools

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/skill"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// skill_read 工具（渐进式披露 L3 内容层，skill_read）：
// agent 根据目录按需读取手册全文，避免把所有手册内容常驻提示词。

// maxSkillContentChars 手册全文返回上限（超出截断，防止撑爆上下文）
const maxSkillContentChars = 20000

type SkillReadParam struct {
	// Code 技能标识（见系统提示词中 Skills 目录）
	Code string `json:"code" jsonschema_description:"技能标识（Skills 目录中反引号内的 code，如 db-ops-guide）"`
}

type SkillReadOutput struct {
	// Code 技能标识
	Code string `json:"code"`
	// Name 技能名称
	Name string `json:"name"`
	// Content 手册全文
	Content string `json:"content"`
}

// GetSkillRead 创建 skill_read 工具（经 agent/ext/skill 的 ToolContributor 通道贡献给 Agent，不再全局注册）
func GetSkillRead() (tool.InvokableTool, error) {
	return utils.InferTool("skill_read",
		"读取指定技能手册的完整内容。当目录中的技能描述与当前任务相关时调用此工具获取详细操作规范。",
		func(ctx context.Context, param *SkillReadParam) (*SkillReadOutput, error) {
			if param.Code == "" {
				return nil, fmt.Errorf("param 'code' is required")
			}
			s, ok := skill.DefaultRegistry.Get(param.Code)
			if !ok {
				return nil, fmt.Errorf("skill not found: %s", param.Code)
			}
			content := s.Content
			if runeLen := len([]rune(content)); runeLen > maxSkillContentChars {
				content = string([]rune(content)[:maxSkillContentChars]) + "\n\n[内容过长已截断]"
			}
			return &SkillReadOutput{
				Code:    s.Code,
				Name:    s.Name,
				Content: content,
			}, nil
		},
	)
}
