package entity

import (
	"mayfly-go/pkg/model"
)

type Machine struct {
	model.Model
	ProjectId          uint64 `json:"projectId"`
	ProjectName        string `json:"projectName"`
	Name               string `json:"name"`
	Ip                 string `json:"ip"`         // IP地址
	Username           string `json:"username"`   // 用户名
	AuthMethod         int8   `json:"authMethod"` // 授权认证方式
	Password           string `json:"-"`
	Port               int    `json:"port"`               // 端口号
	Status             int8   `json:"status"`             // 状态 1:启用；2:停用
	Remark             string `json:"remark"`             // 备注
	EnableSshTunnel    int8   `json:"enableSshTunnel"`    // 是否启用ssh隧道
	SshTunnelMachineId uint64 `json:"sshTunnelMachineId"` // ssh隧道机器id
}

const (
	MachineStatusEnable        int8 = 1  // 启用状态
	MachineStatusDisable       int8 = -1 // 禁用状态
	MachineAuthMethodPassword  int8 = 1  // 密码登录
	MachineAuthMethodPublicKey int8 = 2  // 公钥免密登录
)
