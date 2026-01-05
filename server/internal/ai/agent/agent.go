package agent

import (
	"context"
	"errors"
	"io"
	"mayfly-go/internal/ai/config"
	aimodel "mayfly-go/internal/ai/model"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// GetAiAgent 获取AI Agent
func GetAiAgent(ctx context.Context, aiConfig *config.AIModelConfig, tools ...tool.BaseTool) (*react.Agent, error) {
	aiModel := aimodel.GetAIModelByConfig(aiConfig)
	if aiModel == nil {
		return nil, errors.New("no supported AI model found")
	}
	toolableChatModel, err := aiModel.GetChatModel(ctx, aiConfig)
	if err != nil {
		return nil, err
	}

	// 初始化所需的 tools
	toolsConf := compose.ToolsNodeConfig{
		Tools: tools,
	}
	// 创建 agent
	return react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: toolableChatModel,
		ToolsConfig:      toolsConf,
		MaxStep:          len(toolsConf.Tools)*1 + 3,
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			return input
		},
	})
}

type AiAgent struct {
	*react.Agent
}

// NewAiAgent 创建AI Agent，并注册指定类型的工具
func NewAiAgent(ctx context.Context, toolTypes ...ToolType) (*AiAgent, error) {
	tools := make([]tool.BaseTool, 0)
	for _, toolType := range toolTypes {
		if t, exists := GetTools(toolType); exists {
			tools = append(tools, t...)
		}
	}

	agent, err := GetAiAgent(ctx, config.GetAiModel(), tools...)
	if err != nil {
		return nil, err
	}
	return &AiAgent{
		Agent: agent,
	}, nil
}

// Chat 聊天，返回消息流通道
func (aiAgent *AiAgent) Chat(ctx context.Context, sysPrompt string, question string) (chan *schema.Message, chan error) {
	ch := make(chan *schema.Message, 512)
	errCh := make(chan error, 1)

	if sysPrompt == "" {
		sysPrompt = "你现在是一位拥有20年实战经验的顶级系统运维专家，精通Linux操作系统、数据库管理（如MySQL、PostgreSQL）、NoSQL数据库（如Redis、MongoDB）以及搜索引擎（如Elasticsearch）。"
	}

	agentOption := []agent.AgentOption{}

	go func() {
		defer close(ch)
		defer close(errCh)
		sr, err := aiAgent.Stream(ctx, []*schema.Message{
			{
				Role:    schema.System,
				Content: sysPrompt,
			},
			{
				Role:    schema.User,
				Content: question,
			},
		}, agentOption...)
		if err != nil {
			errCh <- err // 将错误发送到错误通道
			return
		}
		defer sr.Close()

		for {
			msg, err := sr.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				logx.Errorf("failed to recv response: %v", err)
				break
			}
			// logx.Debugf("stream: %s", msg.String())
			ch <- msg
		}
	}()

	return ch, errCh
}

// GetChatMsg 获取完整的聊天回复内容
func (aiAgent *AiAgent) GetChatMsg(ctx context.Context, sysPrompt string, question string) (string, error) {
	msgChan, errChan := aiAgent.Chat(ctx, sysPrompt, question)
	res := ""

	// 使用 select 同时监听消息通道和错误通道
	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// 消息通道已关闭，说明正常结束
				// 检查错误通道是否有错误
				select {
				case err := <-errChan:
					if err != nil {
						return "", err
					}
				default:
					return res, nil
				}
				return res, nil
			}
			res += msg.Content
		case err := <-errChan:
			// 优先检查错误通道
			if err != nil {
				return "", err
			}
		case <-ctx.Done():
			// 上下文被取消
			return "", ctx.Err()
		}
	}
}
