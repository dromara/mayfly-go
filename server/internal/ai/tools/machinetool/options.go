package machinetool

import (
	"context"
	"encoding/json"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
)

// queryMachineOptions 查询当前用户有权限操作的机器列表并格式化为可选项
// （数据源为统一资源查询服务，经「机器+授权凭证」标签做账号级权限过滤）
func queryMachineOptions(ctx context.Context) []tools.CompletionOption {
	return tools.SafeOptionsFn(ctx, "queryMachineOptions", func() []tools.CompletionOption {
		return listMachineOptions(ctx)
	})
}

func listMachineOptions(ctx context.Context) []tools.CompletionOption {
	app := resource.GetApp()
	if app == nil {
		return nil
	}
	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		logx.WarnfContext(ctx, "[queryMachineOptions] no login account in context")
		return nil
	}

	resources, err := app.List(ctx, la.Id, &resource.Query{Types: []string{resource.TypeMachine}})
	if err != nil {
		logx.WarnfContext(ctx, "[queryMachineOptions] failed to list machines: %v", err)
		return nil
	}

	var options []tools.CompletionOption
	for _, r := range resources {
		// 机器的 code 即授权凭证名（machinetool 经 GetCliByAc(code) 建连）
		payload := map[string]any{
			"authCertName": r.Code,
		}
		valueBytes, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		label := r.Name
		if r.Description != "" {
			label += " (" + r.Description + ")"
		}
		options = append(options, tools.CompletionOption{
			Label: label,
			Value: string(valueBytes),
		})
	}

	return options
}
