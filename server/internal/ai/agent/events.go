package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

const (
	// finishReasonLength OpenAI 协议 finish_reason=length：输出被 max_tokens 上限截断
	finishReasonLength = "length"
)

// handleEvents 处理事件
func (a *Agent) handleEvents(ctx context.Context, events *adk.AsyncIterator[*adk.AgentEvent], runOptions *runOptions) ([]adk.Message, error) {
	var outputMessages []adk.Message
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

		var msg adk.Message

		sr := getMessageStream(event)
		if sr != nil {
			// 使用匿名函数或直接在处理完后关闭
			func() {
				defer sr.Close()
				var chunkMessages []adk.Message
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
					if err := runOptions.CallOnChunk(ctx, chunk); err != nil {
						logx.WarnfContext(ctx, "onStreaming callback error: %v", err)
						streamErr = fmt.Errorf("stream callback aborted: %w", err)
						break
					}
				}
				if len(chunkMessages) > 0 {
					// 拼接chunk为完整的消息
					if message, err := schema.ConcatMessages(chunkMessages); err != nil {
						logx.WarnfContext(ctx, "concat streamed messages error: %v", err)
					} else {
						msg = message
					}
				}
			}()
		} else {
			msg = getMessage(event)
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
		if msg.ResponseMeta != nil && msg.ResponseMeta.FinishReason == finishReasonLength {
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

// handleInterrupt 处理中断
func (a *Agent) handleInterrupt(interruptInfo *adk.InterruptInfo) ([]adk.Message, error) {
	outputMessages := []adk.Message{}
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

		extra := NewInternalMessageExtra(string(interruptType), info)
		internalMsg := &schema.Message{
			Role:       session.RoleInternal,
			Content:    info.GetDescription(),
			ToolName:   info.GetToolInfo().Name,
			ToolCallID: toolCallId,
			Extra:      extra,
		}
		SetActionId(internalMsg, ic.ID)
		SetToolStatus(internalMsg, tools.ToolStatusInterrupted)
		outputMessages = append(outputMessages, internalMsg)
	}

	return outputMessages, nil
}

func getMessageStream(event *adk.AgentEvent) adk.MessageStream {
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

func getMessage(event *adk.AgentEvent) *schema.Message {
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
