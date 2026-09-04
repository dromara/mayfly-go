package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

const (
	// finishReasonLength OpenAI 协议 finish_reason=length：输出被 max_tokens 上限截断
	finishReasonLength = "length"
)

// handleEvents 处理事件
//
// eino 事件流中的消息为 AgenticMessage（Typed API 事件边界），消费前统一经
// FromAgenticMessage 解构为 session.Message（全链路统一内存结构）下发回调与收集。
func (a *Agent) handleEvents(ctx context.Context, events *adk.AsyncIterator[*agentEvent], runOptions *runOptions) ([]*session.Message, error) {
	var outputMessages []*session.Message
	var err error
	// 流式接收中断（非 EOF）：部分输出保留但整轮以失败终态收尾；
	// 此前仅 warn 吞错，部分输出被当完整结果且 turn 置 success，
	// 表现为「回复一半静默结束、无任何错误提示」
	var streamErr error

	for {
		event, ok := events.Next()
		if !ok {
			break
		}

		err = event.Err
		if err != nil {
			break
		}

		var msg *session.Message

		sr := getMessageStream(event)
		if sr != nil {
			// 使用匿名函数或直接在处理完后关闭
			func() {
				defer sr.Close()
				var chunkMessages []*schema.AgenticMessage
				for {
					chunk, err := sr.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						logx.WarnfContext(ctx, "stream recv error: %v", err)
						streamErr = fmt.Errorf("stream interrupted: %w", err)
						break
					}
					chunkMessages = append(chunkMessages, chunk)
					if err := runOptions.CallOnChunk(ctx, session.FromAgenticMessage(chunk)); err != nil {
						logx.WarnfContext(ctx, "onStreaming callback error: %v", err)
						streamErr = fmt.Errorf("stream callback aborted: %w", err)
						break
					}
				}
				if len(chunkMessages) > 0 {
					// 拼接chunk为完整的消息
					if message, err := schema.ConcatAgenticMessages(chunkMessages); err != nil {
						logx.WarnfContext(ctx, "concat streamed messages error: %v", err)
					} else {
						msg = session.FromAgenticMessage(message)
					}
				}
			}()
		} else {
			msg = session.FromAgenticMessage(getMessage(event))
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			// 置位中断挂起标记：轮次收尾时结束原因细分为 TurnEndInterrupted
			runOptions.interrupted = true
			interruptInfo := event.Action.Interrupted
			if interruptMessages, err := a.handleInterrupt(interruptInfo); err != nil {
				logx.ErrorfContext(ctx, "interrupt error: %v", err)
				continue
			} else {
				outputMessages = append(outputMessages, interruptMessages...)
				for _, msg := range interruptMessages {
					SetTurnId(msg, runOptions.turnId)
					if err := runOptions.CallOnEvent(ctx, event, msg); err != nil {
						logx.WarnfContext(ctx, "onEvent callback error: %v", err)
						break
					}
				}
			}
		}

		if msg == nil {
			continue
		}
		// 截断检测（流式/非流式统一）：finish_reason=length 表示输出被 max_tokens
		// 上限截断，流正常结束无错误，若不检测会被当完整结果静默收尾——
		// thinking 模型的 reasoning 计入 max_tokens 预算，预算耗尽时
		// tool_call 尚未生成即中断，表现为「回复一半静默停止、无工具调用」
		if finishReasonOf(getAgenticMessage(event)) == finishReasonLength {
			logx.WarnfContext(ctx, "response truncated by max tokens limit")
			streamErr = fmt.Errorf("response truncated: reached the max output tokens limit (maxTokens), please increase the model maxTokens config")
		}
		SetTurnId(msg, runOptions.turnId)

		outputMessages = append(outputMessages, msg)
		if err := runOptions.CallOnEvent(ctx, event, msg); err != nil {
			logx.WarnfContext(ctx, "onEvent callback error: %v", err)
			return outputMessages, err
		}

		LogEventAndMsg(ctx, event, msg)
	}

	// 流中断优先级低于事件循环错误（后者更能表征根因）
	if err == nil {
		err = streamErr
	}
	return outputMessages, err
}

// finishReasonOf 从事件消息的组件扩展中读取 finish_reason（AgenticMessage 路径）
//
// agenticopenai 将 finish_reason 归档到 ResponseMeta.Extension
// （*agenticopenai.ChatResponseMetaExtension）；组件未上报或类型不符返回空。
func finishReasonOf(msg *schema.AgenticMessage) string {
	if msg == nil || msg.ResponseMeta == nil {
		return ""
	}
	if ext, ok := msg.ResponseMeta.Extension.(*agenticopenai.ChatResponseMetaExtension); ok {
		return ext.FinishReason
	}
	return ""
}

// getAgenticMessage 获取事件的完整消息（非流式路径）
func getAgenticMessage(event *agentEvent) *schema.AgenticMessage {
	eo := event.Output
	if eo == nil {
		return nil
	}
	mo := eo.MessageOutput
	if mo == nil {
		return nil
	}
	return mo.Message
}

// handleInterrupt 处理中断
func (a *Agent) handleInterrupt(interruptInfo *adk.InterruptInfo) ([]*session.Message, error) {
	outputMessages := []*session.Message{}
	for _, ic := range interruptInfo.InterruptContexts {
		if !ic.IsRootCause {
			continue
		}

		info, ok := ic.Info.(tools.InterruptMetadata)
		if !ok {
			continue
		}

		interruptType := info.GetType()
		toolCallId := info.GetToolCallId()

		internalMsg := &session.Message{
			Role:       session.RoleInternal,
			Content:    info.GetDescription(),
			ToolName:   info.GetToolInfo().Name,
			ToolCallId: toolCallId,
			Extra:      NewInternalMessageExtra(string(interruptType), info),
		}
		SetActionId(internalMsg, ic.ID)
		SetToolStatus(internalMsg, tools.ToolStatusInterrupted)
		outputMessages = append(outputMessages, internalMsg)
	}

	return outputMessages, nil
}

func getMessageStream(event *agentEvent) adk.AgenticMessageStream {
	eo := event.Output
	if eo == nil {
		return nil
	}
	mo := eo.MessageOutput
	if mo == nil {
		return nil
	}
	return mo.MessageStream
}

func getMessage(event *agentEvent) *schema.AgenticMessage {
	eo := event.Output
	if eo == nil {
		return nil
	}
	mo := eo.MessageOutput
	if mo == nil {
		return nil
	}
	return mo.Message
}
