package agent

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func LogEventAndMsg(ctx context.Context, event *agentEvent, msg *session.Message) {
	agentTag := fmt.Sprintf("Agent - [%s|%s]", event.AgentName, event.RunPath)

	// 定义统一的消息内容
	var eventMsg strings.Builder

	// 思考内容
	if msg.ReasoningContent != "" {
		eventMsg.WriteString(fmt.Sprintf("[THINK] %s\n", msg.ReasoningContent))
	}

	// 消息内容
	if len(msg.Content) > 0 {
		if msg.Role == schema.Tool {
			eventMsg.WriteString(fmt.Sprintf("[TOOL-RESP] %s: %s\n", msg.ToolName, stringx.Truncate(msg.Content, 500, 300, "...")))
		} else {
			eventMsg.WriteString(fmt.Sprintf("[ANSWER] %s\n", stringx.Truncate(msg.Content, 500, 300, "...")))
		}
	}

	// 工具调用
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			eventMsg.WriteString(fmt.Sprintf("[TOOL-CALL] %s(%s)\n", tc.Function.Name, stringx.Truncate(tc.Function.Arguments, 500, 300, "...")))
		}
	}

	// 动作信息
	if event.Action != nil {
		if event.Action.TransferToAgent != nil {
			eventMsg.WriteString(fmt.Sprintf("[TRANSFER] %s\n", event.Action.TransferToAgent.DestAgentName))
		}
		if event.Action.Interrupted != nil {
			for _, ic := range event.Action.Interrupted.InterruptContexts {
				if str, ok := ic.Info.(fmt.Stringer); ok {
					eventMsg.WriteString(fmt.Sprintf("[INTERRUPT] %s\n", str.String()))
				} else {
					eventMsg.WriteString(fmt.Sprintf("[INTERRUPT] %v\n", ic.Info))
				}
			}
		}
		if event.Action.Exit {
			eventMsg.WriteString("[DONE]\n")
		}
	}

	// 错误信息
	if event.Err != nil {
		eventMsg.WriteString(fmt.Sprintf("[ERROR] %v\n", event.Err))
	}

	// 统一输出日志
	finalMsg := eventMsg.String()
	if finalMsg != "" {
		logx.InfofContext(ctx, "%s\n%s", agentTag, finalMsg)
	}
	// return msg, nil
}
