package machinetool

import (
	"errors"

	"github.com/cloudwego/eino/components/tool"
)

// Tools 创建机器工具（聚合创建错误，由装配方统一处理）
//
// 不再注册进全局注册中心：工具经 agent/ext/machinetool 的 ToolContributor
// 通道贡献给 Agent（业务插件注册同名工具即可覆盖，后注册胜出）。
func Tools() ([]tool.BaseTool, error) {
	commandExecTool, err := GetCommandExec()
	if err != nil {
		return nil, errors.Join(errors.New("agent tool - get MachineCommandExec failed"), err)
	}
	return []tool.BaseTool{commandExecTool}, nil
}
