package form

import (
	tagentity "mayfly-go/internal/tag/domain/entity"
)

type InstanceForm struct {
	Id                 uint64  `json:"id"`
	Name               string  `binding:"required" json:"name"`
	Host               string  `binding:"required" json:"host"`
	Port               int     `binding:"required" json:"port"`
	Version            string  `json:"version"`
	Remark             *string `json:"remark"`
	SshTunnelMachineId int     `json:"sshTunnelMachineId"`

	AuthCerts    []*tagentity.ResourceAuthCert `json:"authCerts"` // 资产授权凭证信息列表
	TagCodePaths []string                      `binding:"required" json:"tagCodePaths"`
}
