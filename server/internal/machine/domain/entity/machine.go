package entity

import (
	"mayfly-go/pkg/model"
)

type Machine struct {
	model.Model
	model.ExtraData

	Code               string `json:"code" gorm:"size:32;comment:code"`                      // code
	Name               string `json:"name" gorm:"size:32"`                                   // 名称
	Protocol           int    `json:"protocol" gorm:"default:1;comment:连接协议 1.ssh  2.rdp"`   // 连接协议 1.ssh  2.rdp
	Ip                 string `json:"ip" gorm:"not null;size:100;comment:IP地址"`              // IP地址
	Port               int    `json:"port" gorm:"not null;comment:端口号"`                      // 端口号
	Status             int8   `json:"status" gorm:"not null;default:1;comment:状态 1:启用；2:停用"` // 状态 1:启用；2:停用
	Remark             string `json:"remark" gorm:"comment:备注"`                              // 备注
	SshTunnelMachineId int    `json:"sshTunnelMachineId" gorm:"comment:ssh隧道机器id"`           // ssh隧道机器id
	EnableRecorder     int8   `json:"enableRecorder" gorm:"comment:是否启用终端回放记录"`              // 是否启用终端回放记录
}

const (
	MachineStatusEnable  int8 = 1  // 启用状态
	MachineStatusDisable int8 = -1 // 禁用状态

	MachineProtocolSsh = 1
	MachineProtocolRdp = 2
	MachineProtocolVnc = 3
)
