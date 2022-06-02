package entity

import "mayfly-go/pkg/model"

type MachineFile struct {
	model.Model
	Name string `json:"name"`
	// 机器id
	MachineId uint64 `json:"machineId"`
	Type      int    `json:"type"`
	// 路径
	Path string `json:"path"`
}
