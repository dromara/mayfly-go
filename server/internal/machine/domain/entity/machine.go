package entity

import (
	"fmt"
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/model"
)

type Machine struct {
	model.Model

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
	EnableRecorder     int8   `json:"enableRecorder"`     // 是否启用终端回放记录
	TagId              uint64 `json:"tagId"`
	TagPath            string `json:"tagPath"`
}

const (
	MachineStatusEnable        int8 = 1  // 启用状态
	MachineStatusDisable       int8 = -1 // 禁用状态
	MachineAuthMethodPassword  int8 = 1  // 密码登录
	MachineAuthMethodPublicKey int8 = 2  // 公钥免密登录
)

func (m *Machine) PwdEncrypt() {
	// 密码替换为加密后的密码
	m.Password = utils.PwdAesEncrypt(m.Password)
}

func (m *Machine) PwdDecrypt() {
	// 密码替换为解密后的密码
	m.Password = utils.PwdAesDecrypt(m.Password)
}

// 获取记录日志的描述
func (m *Machine) GetLogDesc() string {
	return fmt.Sprintf("Machine[id=%d, tag=%s, name=%s, ip=%s:%d]", m.Id, m.TagPath, m.Name, m.Ip, m.Port)
}
