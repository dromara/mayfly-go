package dbtool

import (
	"errors"

	"github.com/cloudwego/eino/components/tool"
)

// toolFactory 工具工厂定义（名称用于错误定位）
type toolFactory struct {
	name    string
	factory func() (tool.InvokableTool, error)
}

// Tools 创建全部数据库工具（聚合创建错误，由装配方统一处理）
//
// 不再注册进全局注册中心：工具经 agent/ext/dbtool 的 ToolContributor
// 通道贡献给 Agent（业务插件注册同名工具即可覆盖，后注册胜出）。
func Tools() ([]tool.BaseTool, error) {
	factories := []toolFactory{
		{"QueryTableDDL", GetQueryTableDDL},
		{"QueryTables", GetQueryTables},
		{"QueryData", GetQueryData},
		{"ExecSql", GetSqlExec},
	}

	toolList := make([]tool.BaseTool, 0, len(factories))
	for _, f := range factories {
		t, err := f.factory()
		if err != nil {
			return nil, errors.Join(errors.New("agent tool - get "+f.name+" failed"), err)
		}
		toolList = append(toolList, t)
	}
	return toolList, nil
}
