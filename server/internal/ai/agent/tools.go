package agent

import "github.com/cloudwego/eino/components/tool"

type ToolType string

const (
	ToolTypeDb   ToolType = "db"
	ToolTypeFlow ToolType = "flow"
)

// toolRegistry 工具注册中心，一个ToolType对应多个工具
var toolRegistry = make(map[ToolType][]tool.BaseTool)

// RegisterTool 注册agent工具
func RegisterTool(toolType ToolType, tool ...tool.BaseTool) {
	toolRegistry[toolType] = append(toolRegistry[toolType], tool...)
}

// GetTools 获取指定类型的所有工具
func GetTools(toolType ToolType) ([]tool.BaseTool, bool) {
	tools, exists := toolRegistry[toolType]
	return tools, exists
}

// GetAllTools 获取所有已注册的工具
func GetAllTools() map[ToolType][]tool.BaseTool {
	return toolRegistry
}

// RegisterTools 批量注册工具
func RegisterTools(tools map[ToolType][]tool.BaseTool) {
	for toolType, toolList := range tools {
		toolRegistry[toolType] = append(toolRegistry[toolType], toolList...)
	}
}

// GetToolsByTypes 获取指定类型的多个工具
func GetToolsByTypes(types []ToolType) map[ToolType][]tool.BaseTool {
	result := make(map[ToolType][]tool.BaseTool)
	for _, t := range types {
		if tools, exists := toolRegistry[t]; exists {
			result[t] = tools
		}
	}
	return result
}

// GetFirstTool 获取指定类型的第一个工具（常用场景）
func GetFirstTool(toolType ToolType) (tool.BaseTool, bool) {
	tools, exists := toolRegistry[toolType]
	if !exists || len(tools) == 0 {
		return nil, false
	}
	return tools[0], true
}

// ClearTools 清空指定类型的工具
func ClearTools(toolType ToolType) {
	delete(toolRegistry, toolType)
}
