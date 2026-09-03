package machinetool

import (
	"context"
	"encoding/json"
	"mayfly-go/internal/ai/tools"
	machineapp "mayfly-go/internal/machine/application"
	machineentity "mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

// queryMachineOptions 查询可用机器列表并格式化为可选项
func queryMachineOptions(ctx context.Context) []tools.CompletionOption {
	return tools.SafeOptionsFn(ctx, "queryMachineOptions", func() []tools.CompletionOption {
		return listMachineOptions(ctx)
	})
}

func listMachineOptions(ctx context.Context) []tools.CompletionOption {
	machineApp := machineapp.GetMachineApp()
	if machineApp == nil {
		return nil
	}

	pageResult, err := machineApp.GetMachineList(&machineentity.MachineQuery{
		PageParam: model.PageParam{PageNum: 1, PageSize: 100},
		Status:    machineentity.MachineStatusEnable,
	})
	if err != nil {
		logx.WarnfContext(ctx, "[queryMachineOptions] failed to list machines: %v", err)
		return nil
	}

	var options []tools.CompletionOption
	for _, m := range pageResult.List {
		if m == nil {
			continue
		}
		label := m.Name
		if m.Ip != "" {
			label += " (" + m.Ip + ")"
		}
		payload := map[string]any{
			"authCertName": m.Code,
		}
		valueBytes, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		options = append(options, tools.CompletionOption{
			Label: label,
			Value: string(valueBytes),
		})
	}

	return options
}
