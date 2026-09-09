// Package skillext 技能手册目录扩展（渐进式披露，skill_injection.rs）
//
// 实现 ContextContributor 通道：L1 目录（本轮 preamble，预算控制）+
// L3 内容（agent 按需调用 skill_read 工具获取全文）。
package skillext

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/skill"
	"mayfly-go/pkg/logx"
	"regexp"
)

// 技能目录预算控制
const (
	// maxSkillDescriptionChars 单个技能描述最大字符数
	maxSkillDescriptionChars = 1024
	// skillCatalogBudgetChars 技能目录总预算（字符数）
	skillCatalogBudgetChars = 8000
)

// explicitSkillMentionRegexp 匹配用户输入中显式提及的技能 code（$skill-code 语法）
var explicitSkillMentionRegexp = regexp.MustCompile(`\$([a-zA-Z0-9][a-zA-Z0-9_-]*)`)

// SkillCatalogExtension 技能手册目录注入扩展（L1 索引层）
type SkillCatalogExtension struct{}

// NewExtension 创建技能目录扩展
func NewExtension() *SkillCatalogExtension {
	return &SkillCatalogExtension{}
}

var _ contributor.ContextContributor = (*SkillCatalogExtension)(nil)

func (e *SkillCatalogExtension) Id() string     { return "skill_injection" }
func (e *SkillCatalogExtension) Retained() bool { return false }
func (e *SkillCatalogExtension) Priority() int  { return 50 }

func (e *SkillCatalogExtension) ContributeTurnContext(ctx context.Context, in *contributor.TurnInput) ([]contributor.PromptFragment, error) {
	content := RenderSkillCatalog(in.UserText)
	if content == "" {
		return nil, nil
	}
	logx.DebugfContext(ctx, "[skill_injection] injecting %d chars of skill catalog", len(content))
	return []contributor.PromptFragment{
		{Source: "skill_injection", Content: content},
	}, nil
}

// RenderSkillCatalog 渲染技能目录（带 $code 显式提及标记与预算控制）
//
// 无已注册技能时返回空串（调用方跳过注入）。
func RenderSkillCatalog(userText string) string {
	skills := skill.DefaultRegistry.List()
	if len(skills) == 0 {
		return ""
	}

	// 显式技能来源：文本 $code 提及
	explicitCodes := make(map[string]struct{})
	for _, code := range extractExplicitSkillCodes(userText, skill.DefaultRegistry.Exists) {
		explicitCodes[code] = struct{}{}
	}

	output := "## Skills\n"
	output += "A skill is a set of instructions provided through a manual source.\n"
	output += "Use the `skill_read` tool to load the full content of a skill when needed.\n\n"
	output += "### Available skills\n"

	lines := make([]string, 0, len(skills))
	for _, s := range skills {
		line := fmt.Sprintf("- `%s`: %s", s.Code, s.Name)
		if s.Description != "" {
			line += " - " + truncateSkillDescription(s.Description, maxSkillDescriptionChars)
		}
		if _, ok := explicitCodes[s.Code]; ok {
			line += " ⭐ *[explicitly mentioned]*"
		}
		lines = append(lines, line)
	}

	// 预算控制：超出总预算时省略放不下的技能并加标记
	omitted := 0
	included := make([]string, 0, len(lines))
	current := runeCount(output)
	for _, line := range lines {
		lineChars := runeCount(line) + 1
		if current+lineChars <= skillCatalogBudgetChars {
			included = append(included, line)
			current += lineChars
		} else {
			omitted++
		}
	}

	for _, line := range included {
		output += line + "\n"
	}
	if omitted > 0 {
		output += fmt.Sprintf("\n- %d additional skills omitted to fit the context limit.\n", omitted)
	}
	return output
}

// extractExplicitSkillCodes 从用户输入中提取显式提及且已注册的技能 code
func extractExplicitSkillCodes(userText string, exists func(string) bool) []string {
	if userText == "" {
		return nil
	}
	matches := explicitSkillMentionRegexp.FindAllStringSubmatch(userText, -1)
	codes := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 && exists(m[1]) {
			codes = append(codes, m[1])
		}
	}
	return codes
}

// truncateSkillDescription 基于字符数截断（UTF-8 安全），超长时添加 "..." 后缀
func truncateSkillDescription(description string, maxChars int) string {
	runes := []rune(description)
	if len(runes) <= maxChars {
		return description
	}
	prefix := maxChars - 3
	return string(runes[:prefix]) + "..."
}

func runeCount(s string) int { return len([]rune(s)) }

// Install 注册扩展到 Builder
func Install(b *contributor.Builder) {
	b.RegisterContext(NewExtension())
}
