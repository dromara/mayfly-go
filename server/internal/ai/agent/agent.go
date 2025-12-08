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
func GetAiAgent(ctx context.Context, aiConfig *config.AIModelConfig) (*react.Agent, error) {
	aiModel := aimodel.GetAIModelByConfig(aiConfig)
	if aiModel == nil {
		return nil, errors.New("no supported AI model found")
	}
	toolableChatModel, err := aiModel.GetChatModel(ctx, aiConfig)
	if err != nil {
		return nil, err
	}
	// 初始化所需的 tools
	aiTools := compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{},
	}
	// 创建 agent
	return react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: toolableChatModel,
		ToolsConfig:      aiTools,
		MaxStep:          len(aiTools.Tools)*1 + 3,
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			return input
		},
	})
}

type AiAgent struct {
	*react.Agent
}

// NewAiAgent 创建AI Agent
func NewAiAgent(ctx context.Context) (*AiAgent, error) {
	agent, err := GetAiAgent(ctx, config.GetAiModel())
	if err != nil {
		return nil, err
	}
	return &AiAgent{
		Agent: agent,
	}, nil
}

// Chat 聊天，返回消息流通道
func (aiAgent *AiAgent) Chat(ctx context.Context, sysPrompt string, question string) chan *schema.Message {
	ch := make(chan *schema.Message, 512)

	if sysPrompt == "" {
		sysPrompt = "你现在是一位拥有20年实战经验的顶级系统运维专家，精通Linux操作系统、数据库管理（如MySQL、PostgreSQL）、NoSQL数据库（如Redis、MongoDB）以及搜索引擎（如Elasticsearch）。"
	}

	agentOption := []agent.AgentOption{}

	go func() {
		defer close(ch)
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
			logx.Errorf("agent stream error: %v", err)
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

	return ch
}

// GetChatMsg 获取完整的聊天回复内容
func (aiAgent *AiAgent) GetChatMsg(ctx context.Context, sysPrompt string, question string) string {
	msgChan := aiAgent.Chat(ctx, sysPrompt, question)
	res := ""
	for msg := range msgChan {
		res += msg.Content
	}
	return res
}
