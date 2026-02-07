package agent

import (
	"context"
	"mayfly-go/internal/ai/config"
	aimodel "mayfly-go/internal/ai/model"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// GetAgent 获取AI Agent
func GetAgent(ctx context.Context, aiConfig *config.AIModelConfig, tools ...tool.BaseTool) (adk.Agent, error) {
	toolableChatModel, err := aimodel.GetChatModel(ctx, aiConfig)
	if err != nil {
		return nil, err
	}

	// 初始化所需的 tools
	toolsConfig := adk.ToolsConfig{}
	toolsConfig.Tools = tools

	chatAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ops_expert",
		Description: "一位拥有20多年系统管理、数据库管理和基础设施优化经验的专业DevOps专家。",
		Instruction: `你现在是一位专业的数据库管理员、Redis管理员和安全审核专家，请根据用户的问题给出最合适的答案。`,
		Model:       toolableChatModel,
		ToolsConfig: toolsConfig,
	})
	if err != nil {
		return nil, err
	}

	return chatAgent, nil
}

// GetOpsExpertAgent 获取运维专家agent
func GetOpsExpertAgent(ctx context.Context, toolTypes ...ToolType) (*AiAgent, error) {
	tools := make([]tool.BaseTool, 0)
	for _, toolType := range toolTypes {
		if t, exists := GetTools(toolType); exists {
			tools = append(tools, t...)
		}
	}

	agent, err := GetAgent(ctx, config.GetAiModel(), tools...)
	if err != nil {
		return nil, err
	}
	return &AiAgent{
		Agent: agent,
	}, nil
}

type AiAgent struct {
	adk.Agent
}

// Run 运行，并返回最终结果
func (aiAgent *AiAgent) Run(ctx context.Context, sysPrompt string, question string) (string, error) {
	if sysPrompt == "" {
		sysPrompt = "你现在是一位拥有20年实战经验的顶级系统运维专家，精通Linux操作系统、数据库管理（如MySQL、PostgreSQL）、NoSQL数据库（如Redis、MongoDB）以及搜索引擎（如Elasticsearch）。"
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           aiAgent.Agent,
		CheckPointStore: NewInMemoryStore(),
	})

	iter := runner.Run(ctx, []adk.Message{
		{
			Role:    schema.System,
			Content: sysPrompt,
		},
		{
			Role:    schema.User,
			Content: question,
		},
	})

	res := ""
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		err := event.Err
		if err != nil {
			logx.Error(err.Error())
			return res, err
		}

		LogEvent(event)
		msg := event.Output.MessageOutput.Message
		if msg != nil {
			res = msg.Content
		}
	}

	return res, nil
}
