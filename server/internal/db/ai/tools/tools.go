package tools

import (
	"mayfly-go/internal/ai/agent"
	"mayfly-go/pkg/logx"
)

func Init() {
	if queryTableTool, err := GetQueryTableInfo(); err != nil {
		logx.Errorf("agent tool - 获取QueryTableInfo工具失败: %v", err)
	} else {
		agent.RegisterTool(agent.ToolTypeDb, queryTableTool)
	}
}
