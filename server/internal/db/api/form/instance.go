package form

import (
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
)

type InstanceForm struct {
	model.ExtraData

	Id                 uint64 `json:"id"`
	Name               string `binding:"required" json:"name"`
	Type               string `binding:"required" json:"type"` // 类型，mysql oracle等
	Host               string `binding:"required" json:"host"`
	Port               int    `json:"port"`
	Params             string `json:"params"`
	Remark             string `json:"remark"`
	SshTunnelMachineId int    `json:"sshTunnelMachineId"`

	AuthCerts    []*tagentity.ResourceAuthCert `json:"authCerts" binding:"required"` // 资产授权凭证信息列表
	TagCodePaths []string                      `binding:"required" json:"tagCodePaths"`
}

type InstanceDbNamesForm struct {
	model.ExtraData

	Type               string                      `binding:"required" json:"type"` // 类型，mysql oracle等
	Host               string                      `binding:"required" json:"host"`
	Port               int                         `json:"port"`
	Params             string                      `json:"params"`
	SshTunnelMachineId int                         `json:"sshTunnelMachineId"`
	AuthCert           *tagentity.ResourceAuthCert `json:"authCert" binding:"required"` // 资产授权凭证信息
}
