package models

import "mayfly-go/base/model"

type MachineScript struct {
	model.Model
	Name string `json:"name"`
	// 机器id
	MachineId uint64 `json:"machineId"`
	Type      int    `json:"type"`
	// 脚本内容
	Description string `json:"description"`
	// 脚本内容
	Script string `json:"script"`
}
