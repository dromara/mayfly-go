package entity

import (
	"mayfly-go/pkg/model"
)

type Machine struct {
	model.Model
	ProjectId   uint64 `json:"projectId"`
	ProjectName string `json:"projectName"`
	Name        string `json:"name"`
	Ip          string `json:"ip"`       // IP地址
	Username    string `json:"username"` // 用户名
	Password    string `json:"-"`
	Port        int    `json:"port"`   // 端口号
	Status      int8   `json:"status"` // 状态 1:启用；2:停用
	Remark      string `json:"remark"` // 备注
}

const (
	MachineStatusEnable  int8 = 1  // 启用状态
	MachineStatusDisable int8 = -1 // 禁用状态
)
