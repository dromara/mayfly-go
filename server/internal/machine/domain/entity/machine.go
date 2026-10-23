package entity

import (
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
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

	// 机器扩展配置（extra）中的出网代理键：保存/装配/接口脱敏共用这一组键名
	ExtraKeyProxyType          = "proxyType"
	ExtraKeyProxyAddr          = "proxyAddr"
	ExtraKeyProxyUsername      = "proxyUsername"
	ExtraKeyProxyPassword      = "proxyPassword"
	ExtraKeyProxyPasswordClear = "proxyPasswordClear"
	// MachineProxyTypeNone proxyType 取值：显式直连
	MachineProxyTypeNone = "none"
	// MachineProxyTypeSocks5 proxyType 取值：SOCKS5 代理
	MachineProxyTypeSocks5 = "socks5"
)

// CloneExtraWithoutProxyCredential 返回不含代理凭据的 extra 副本。
// 审计日志、下发前脱敏等场景必须基于副本操作：表单与实体在 BindJsonAndCopyTo 后共享同一个 map，
// 直接删键会连带删掉待加密落库的明文密码（实测导致“保存成功但密码未存”）。
func CloneExtraWithoutProxyCredential(extra model.ExtraData) model.ExtraData {
	if len(extra.Extra) == 0 {
		return model.ExtraData{}
	}
	cloned := make(collx.M, len(extra.Extra))
	for k, v := range extra.Extra {
		cloned[k] = v
	}
	delete(cloned, ExtraKeyProxyPassword)
	delete(cloned, ExtraKeyProxyPasswordClear)
	return model.ExtraData{Extra: cloned}
}
