package application

import (
	"strings"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/ai/protocol"
)

// instructionText 任务/触发器指令 → 真正发给模型的提示词。
//
// 有芯片引用段时按 ai 协议渲染器就地展开：技能注入技能码（供 skill_read 加载），资源注入
// machineId/machineIp/dbId/dbName 等完整定位信息，使工具入参不必靠模型猜（此前实测出现过
// 「凭空猜资源 id / 把资源显示名当 schema 名」导致工具报错中断整轮运行）。
// 没有段时按原文下发——内置 Runbook 模板与历史任务都是纯文本指令，行为不变。
//
// 渲染器与聊天消息共用同一注册表（protocol.RegisterChipRenderer），因此「在助手里引用同一资源」
// 与「在任务指令里引用同一资源」喂给模型的文本口径一致，不会有两套展开规则
func instructionText(text string, segments []entity.InstructionSegment) string {
	if len(segments) == 0 {
		return text
	}
	protocolSegments := make([]protocol.ContentSegment, 0, len(segments))
	for _, s := range segments {
		protocolSegments = append(protocolSegments, protocol.ContentSegment{Type: s.Type, Text: s.Text, Extra: s.Extra})
	}
	rendered := protocol.RenderInstruction(protocolSegments)
	// 段与原文必须同源（编辑器同时产出两者：文本段拼起来就是指令原文）。不同源说明客户端只发了
	// 一半——此时以用户写的原文为准，引用段附后展开，绝不能让引用段反过来吞掉指令
	if strings.TrimSpace(segmentText(segments)) != strings.TrimSpace(text) {
		return strings.TrimSpace(text + "\n" + rendered)
	}
	return rendered
}

// segmentText 拼出段里的全部文本（用于上面的同源性判定）
func segmentText(segments []entity.InstructionSegment) string {
	var sb strings.Builder
	for _, s := range segments {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

// hasResourceRef 指令段里是否引用了具体资源。
//
// 用于决定运行输入要不要显式引导「先 ListResources 盘点」：目标资源不再是独立字段后，
// 「这次对哪台机器/哪个库」唯一的来源就是指令里的 @资源引用，没引用才需要让模型先盘点
func hasResourceRef(segments []entity.InstructionSegment) bool {
	for _, seg := range segments {
		if seg.Type == protocol.ContentSegmentResource {
			return true
		}
	}
	return false
}
