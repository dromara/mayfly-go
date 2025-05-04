package entity

import (
	"mayfly-go/pkg/model"
)

// MachineCmdConf 机器命令过滤配置
type MachineCmdConf struct {
	model.Model

	Name     string              `json:"name" gorm:"size:100;comment:名称"`            // 名称
	Cmds     model.Slice[string] `json:"cmds" gorm:"type:varchar(500);comment:命令配置"` // 命令配置
	Status   int8                `json:"status" gorm:"comment:状态"`                   // 状态
	Stratege string              `json:"stratege" gorm:"size:100;comment:策略"`        // 策略，空禁用
	Remark   string              `json:"remark" gorm:"size:50;comment:备注"`           // 备注
}
