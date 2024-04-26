package entity

import (
	"mayfly-go/pkg/model"
)

// 机器命令过滤配置
type MachineCmdConf struct {
	model.Model

	Name     string              `json:"name"`
	Cmds     model.Slice[string] `json:"cmds"`     // 命令配置
	Status   int8                `json:"execCmds"` // 状态
	Stratege string              `json:"stratege"` // 策略，空禁用
	Remark   string              `json:"remark"`   // 备注
}
