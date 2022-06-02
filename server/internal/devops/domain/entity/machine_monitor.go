package entity

import (
	"time"
)

type MachineMonitor struct {
	Id         uint64    `json:"id"`
	MachineId  uint64    `json:"machineId"`
	CpuRate    float32   `json:"cpuRate"`
	MemRate    float32   `json:"memRate"`
	SysLoad    string    `json:"sysLoad"`
	CreateTime time.Time `json:"createTime"`
}
