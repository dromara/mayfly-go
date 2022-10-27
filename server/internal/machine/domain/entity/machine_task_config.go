package entity

import "mayfly-go/pkg/model"

// 机器任务配置
type MachineTaskConfig struct {
	model.Model

	Name           string `json:"name"`
	Cron           string `json:"cron"`   // cron表达式
	Script         string `json:"script"` // 任务内容
	Status         string `json:"status"`
	EnableNotify   int    `json:"enableNotify"`   // 是否启用通知
	NotifyTemplate string `json:"notifyTemplate"` // 通知模板
	Remark         string `json:"remark"`         // 备注
}
