package protocol

import (
	"fmt"
	"strings"
)

// 芯片段 LLM 注入文本渲染（注册式扩展点，对齐 contributor/中断扩展的装配哲学）
//
// 内容段（ContentSegment）到发送文本的渲染不再由 api 层 switch 硬编码：
// 芯片类型与资源类型均为注册表驱动，新增类型只需注册渲染器，
// RenderSegments 与编排层零修改（开闭原则）。

// ChipRenderer 内容段渲染器：将一个内容段渲染为发给 LLM 的注入文本
//
// 未注册渲染器的段按原文透出（纯文本段天然走此路径）。
type ChipRenderer func(seg ContentSegment) string

var chipRenderers = map[string]ChipRenderer{}

// RegisterChipRenderer 注册芯片段类型渲染器（同类型覆盖，后注册胜出）
func RegisterChipRenderer(segType string, fn ChipRenderer) {
	if fn == nil {
		return
	}
	chipRenderers[segType] = fn
}

// ResourceIdentityRenderer 资源标识渲染器：按资源类型渲染人类可读 typeName
// 与工具调用所需的完整定位标识（id/code/ip/port/凭证等，使 LLM 可直接取参，
// 避免工具调用因参数缺失触发中断补全）
//
// 新增可引用资源类型（redis/mongo/es 等）时注册本渲染器即可，
// 无需修改 resource 段渲染逻辑。
type ResourceIdentityRenderer func(extra map[string]any) (typeName string, identity string)

var resourceIdentityRenderers = map[string]ResourceIdentityRenderer{}

// RegisterResourceIdentityRenderer 注册资源类型标识渲染器（同类型覆盖，后注册胜出）
func RegisterResourceIdentityRenderer(resourceType string, fn ResourceIdentityRenderer) {
	if fn == nil {
		return
	}
	resourceIdentityRenderers[resourceType] = fn
}

// RenderSegments 将结构化内容段渲染为最终发送给 LLM 的文本
//
// 存在芯片段时尾部追加聚焦指令：会话早期可能出现过其它资源的调用失败语境，
// 明确圈定本次任务范围，防止模型被历史上下文带偏；纯文本段不追加。
func RenderSegments(segments []ContentSegment) string {
	if len(segments) == 0 {
		return ""
	}
	var sb strings.Builder
	hasChip := false
	for _, seg := range segments {
		if render, ok := chipRenderers[seg.Type]; ok {
			sb.WriteString(render(seg))
			hasChip = true
			continue
		}
		sb.WriteString(seg.Text)
	}
	if hasChip {
		sb.WriteString("\n（以上引用资源为本次任务的唯一目标，请仅针对该资源执行，忽略会话早期出现的其它资源）")
	}
	return strings.TrimSpace(sb.String())
}

func init() {
	RegisterChipRenderer(ContentSegmentResource, renderResourceSegment)
	RegisterChipRenderer(ContentSegmentSkill, renderSkillSegment)
	RegisterResourceIdentityRenderer("machine", renderMachineIdentity)
	RegisterResourceIdentityRenderer("db", renderDbIdentity)
}

// renderResourceSegment 资源芯片段渲染（标识细节由 resourceIdentityRenderers 按类型分发）
func renderResourceSegment(seg ContentSegment) string {
	resourceType, _ := seg.Extra["resourceType"].(string)
	typeName := resourceType
	identity := ""
	if render, ok := resourceIdentityRenderers[resourceType]; ok {
		typeName, identity = render(seg.Extra)
	} else if id, _ := seg.Extra["id"].(string); id != "" {
		identity = fmt.Sprintf("id=%s", id)
	}
	if identity != "" {
		return fmt.Sprintf("[引用资源] %s: %s (%s)\n", typeName, seg.Text, identity)
	}
	return fmt.Sprintf("[引用资源] %s: %s\n", typeName, seg.Text)
}

// renderSkillSegment 技能芯片段渲染（技能内容经渐进式披露获取：
// L1 目录由 skill_injection 注入系统提示词，L3 全文经 skill_read 工具按需读取）
func renderSkillSegment(seg ContentSegment) string {
	return fmt.Sprintf("[引用技能] %s\n", seg.Text)
}

// renderMachineIdentity 机器资源标识：注入 id/code/ip/port 完整定位；
// 引用选到授权凭证层级时 authCertName/username 一并注入，
// MachineCommandExec 等工具参数直接可取，避免中断等待用户补全
func renderMachineIdentity(extra map[string]any) (string, string) {
	id, _ := extra["id"].(string)
	code, _ := extra["code"].(string)
	ip, _ := extra["ip"].(string)
	port := fmt.Sprintf("%v", extra["port"])
	identity := fmt.Sprintf("id=%s, code=%s, ip=%s, port=%s", id, code, ip, port)
	if acName, _ := extra["authCertName"].(string); acName != "" {
		identity += fmt.Sprintf(", authCertName=%s", acName)
		if username, _ := extra["username"].(string); username != "" {
			identity += fmt.Sprintf(", username=%s", username)
		}
	}
	return "机器", identity
}

// renderDbIdentity 数据库资源标识：引用选到物理库 + 授权账号时
// db/authCertName/username 一并注入，db 工具的 dbId/dbName 参数直接可取
func renderDbIdentity(extra map[string]any) (string, string) {
	id, _ := extra["id"].(string)
	code, _ := extra["code"].(string)
	identity := fmt.Sprintf("id=%s, code=%s", id, code)
	if dbName, _ := extra["db"].(string); dbName != "" {
		identity += fmt.Sprintf(", db=%s", dbName)
	}
	if acName, _ := extra["authCertName"].(string); acName != "" {
		identity += fmt.Sprintf(", authCertName=%s", acName)
	}
	if username, _ := extra["username"].(string); username != "" {
		identity += fmt.Sprintf(", username=%s", username)
	}
	return "数据库", identity
}
