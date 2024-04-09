package entity

import (
	"mayfly-go/pkg/model"
)

type Machine struct {
	model.Model

	Code               string `json:"code"`
	Name               string `json:"name"`
	Protocol           int    `json:"protocol"`           // 连接协议 1.ssh  2.rdp
	Ip                 string `json:"ip"`                 // IP地址
	Port               int    `json:"port"`               // 端口号
	Status             int8   `json:"status"`             // 状态 1:启用；2:停用
	Remark             string `json:"remark"`             // 备注
	SshTunnelMachineId int    `json:"sshTunnelMachineId"` // ssh隧道机器id
	EnableRecorder     int8   `json:"enableRecorder"`     // 是否启用终端回放记录
}

const (
	MachineStatusEnable  int8 = 1  // 启用状态
	MachineStatusDisable int8 = -1 // 禁用状态

	MachineProtocolSsh = 1
	MachineProtocolRdp = 2
)
