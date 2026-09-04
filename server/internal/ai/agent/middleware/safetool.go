package middleware

import (
	"context"
	"errors"
	"mayfly-go/internal/ai/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type SafeToolMiddleware struct {
	*adk.TypedBaseChatModelAgentMiddleware[*schema.AgenticMessage]
}

func (m *SafeToolMiddleware) WrapInvokableToolCall(
	ctx context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tc *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, args, opts...)
		if err == nil {
			return result, nil
		}

		// 中断错误不转换，需要继续传播
		if _, ok := compose.IsInterruptRerunError(err); ok {
			return "", err
		}

		if toolErr, ok := errors.AsType[*tools.ToolError](err); ok {
			// 支持重试策略，将其转换为错误字符串
			if toolErr.Strategy == tools.RecoverRetry {
				return tools.GetToolErrorMsg(err), nil
			}
		}

		return "", tools.NewToolError(err, tools.RecoverNone).WithToolName(tc.Name).WithToolCallId(tc.CallID)
	}, nil
}

func (m *SafeToolMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tc *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		sr, err := endpoint(ctx, args, opts...)
		if err == nil {
			return safeWrapReader(sr), nil
		}

		if _, ok := compose.IsInterruptRerunError(err); ok {
			return nil, err
		}

		if toolErr, ok := errors.AsType[*tools.ToolError](err); ok {
			// 支持重试策略，将其转换为错误字符串
			if toolErr.Strategy == tools.RecoverRetry {
				return singleChunkReader(tools.GetToolErrorMsg(err)), nil
			}
		}

		return nil, tools.NewToolError(err, tools.RecoverNone).WithToolName(tc.Name).WithToolCallId(tc.CallID)
	}, nil
}
