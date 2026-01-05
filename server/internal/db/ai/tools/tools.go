package tools

import "mayfly-go/internal/ai/agent"

func Init() {
	agent.RegisterTool(agent.ToolTypeDb, GetQueryTableInfo())
}
