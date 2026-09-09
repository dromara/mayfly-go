package contributor

import (
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/schema"
)

// 协议完整性校验（Normalize 通道，normalize.rs）
//
// 在历史发送给模型前执行完整性修复，防止 API 400 错误：
//   - EnsureCallOutputsPresent：为缺失 output 的 tool_call 合成 "[aborted]" 占位符
//   - RemoveOrphanOutputs：移除无对应 tool_call 的孤儿 tool 结果
//   - NormalizeHistory：组合通道（先补全 → 再去孤儿）
//
// 合成 output 使用确定性内容（基于 call_id 关联），保持 prompt cache 友好。

// syntheticAbortedOutput 合成 output 的内容模板
const syntheticAbortedOutput = "[aborted] Tool execution was interrupted or cancelled."

// NormalizeHistory 组合 normalize 通道（推荐入口）
//
// 按顺序执行：补全缺失的 tool 结果 → 移除孤儿 tool 结果。
// 纯函数，无副作用，可安全重试（幂等）。
// 由 Registry.CollectHistory 对合并后的全量列表统一执行一次。
func NormalizeHistory(messages []*session.Message) []*session.Message {
	messages = ensureCallOutputsPresent(messages)
	return removeOrphanOutputs(messages)
}

// ensureCallOutputsPresent 为缺失 output 的 tool_call 合成 "[aborted]" 占位符
//
// 场景：用户取消工具执行（中断恢复后部分 call 无 result）、
// 存储中某条 tool_result 误删、压缩/截断过程中丢失 result。
//
// 合成策略：缺失的 result 追加到消息列表末尾（normalize 主要在
// 历史重建场景使用，此时历史已固定，追加是最安全的关联方式）。
func ensureCallOutputsPresent(messages []*session.Message) []*session.Message {
	// 第一遍：收集全部已有 tool 结果 id（与顺序无关）
	existingResults := make(map[string]struct{})
	for _, msg := range messages {
		if msg.Role == schema.Tool && msg.ToolCallId != "" {
			existingResults[msg.ToolCallId] = struct{}{}
		}
	}

	// 第二遍：收集全部缺失结果的 tool_call（按出现顺序）
	var missingCalls []schema.ToolCall
	for _, msg := range messages {
		if msg.Role != schema.Assistant {
			continue
		}
		for _, call := range msg.ToolCalls {
			if _, ok := existingResults[call.ID]; !ok {
				missingCalls = append(missingCalls, call)
			}
		}
	}

	if len(missingCalls) == 0 {
		return messages
	}

	logx.Warnf("[normalize] synthesizing %d aborted tool_result(s) for missing outputs", len(missingCalls))
	for _, call := range missingCalls {
		logx.Warnf("[normalize] tool_call '%s' (%s) has no matching result, synthesizing aborted output", call.ID, call.Function.Name)
		messages = append(messages, &session.Message{
			Role:       schema.Tool,
			Content:    syntheticAbortedOutput,
			ToolCallId: call.ID,
			ToolName:   call.Function.Name,
		})
	}
	return messages
}

// removeOrphanOutputs 移除无对应 tool_call 的孤儿 tool 结果
//
// 场景：压缩/截断移除了 assistant 消息但保留了对应的 tool 结果、
// 存储数据不一致、中断恢复时状态与外部历史不匹配。
func removeOrphanOutputs(messages []*session.Message) []*session.Message {
	callIds := make(map[string]struct{})
	for _, msg := range messages {
		if msg.Role == schema.Assistant {
			for _, call := range msg.ToolCalls {
				callIds[call.ID] = struct{}{}
			}
		}
	}

	orphanCount := 0
	for _, msg := range messages {
		if msg.Role == schema.Tool && msg.ToolCallId != "" {
			if _, ok := callIds[msg.ToolCallId]; !ok {
				orphanCount++
			}
		}
	}
	if orphanCount == 0 {
		return messages
	}

	logx.Warnf("[normalize] removing %d orphan tool_result(s) (no matching tool_call)", orphanCount)
	result := make([]*session.Message, 0, len(messages)-orphanCount)
	for _, msg := range messages {
		if msg.Role == schema.Tool && msg.ToolCallId != "" {
			if _, ok := callIds[msg.ToolCallId]; !ok {
				continue
			}
		}
		result = append(result, msg)
	}
	return result
}
