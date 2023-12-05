package entity

import (
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/model"
)

type Machine struct {
	model.Model

	Code               string `json:"code"`
	Name               string `json:"name"`
	Ip                 string `json:"ip"`                 // IP地址
	Port               int    `json:"port"`               // 端口号
	Username           string `json:"username"`           // 用户名
	Password           string `json:"password"`           // 密码
	AuthCertId         int    `json:"authCertId"`         // 授权凭证id
	Status             int8   `json:"status"`             // 状态 1:启用；2:停用
	Remark             string `json:"remark"`             // 备注
	SshTunnelMachineId int    `json:"sshTunnelMachineId"` // ssh隧道机器id
	EnableRecorder     int8   `json:"enableRecorder"`     // 是否启用终端回放记录
}

const (
	MachineStatusEnable  int8 = 1  // 启用状态
	MachineStatusDisable int8 = -1 // 禁用状态
)

func (m *Machine) PwdEncrypt() {
	// 密码替换为加密后的密码
	m.Password = utils.PwdAesEncrypt(m.Password)
}

func (m *Machine) PwdDecrypt() {
	// 密码替换为解密后的密码
	m.Password = utils.PwdAesDecrypt(m.Password)
}

func (m *Machine) UseAuthCert() bool {
	return m.AuthCertId > 0
}
