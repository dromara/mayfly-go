package entity

import "mayfly-go/pkg/model"

type MachineScript struct {
	model.Model

	Name        string `json:"name" gorm:"not null;size:255;comment:脚本名"`     // 脚本名
	MachineId   uint64 `json:"machineId" gorm:"not null;comment:机器id[0:公共]"`  // 机器id
	Type        int    `json:"type" gorm:"comment:脚本类型[1: 有结果；2：无结果；3：实时交互]"` // 脚本类型[1: 有结果；2：无结果；3：实时交互]
	Category    string `json:"category" gorm:"size:20;comment:分类"`
	Description string `json:"description" gorm:"size:255;comment:脚本描述"` // 脚本描述
	Params      string `json:"params" gorm:"size:500;comment:脚本入参"`      // 参数列表json
	Script      string `json:"script" gorm:"type:text;comment:脚本内容"`     // 脚本内容
}
