package form

import (
	"mayfly-go/internal/pkg/tlsx"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
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

	// TlsMaterial 表单明文提交的证书材料；档位/主机名/是否用系统 CA 走 extra（可读配置）。
	// 编辑态留空 = 沿用库中旧密文（表单从不回显证书内容）。
	TlsMaterial *tlsx.Material `json:"tlsMaterial"`

	AuthCerts    []*tagentity.ResourceAuthCert `json:"authCerts" binding:"required"` // 资产授权凭证信息列表
	TagCodePaths []string                      `binding:"required" json:"tagCodePaths"`
}

// AuditReplica 生成用于操作日志的表单副本。
//
// 必须另建对象而不是直接记 form：表单里带着授权凭证的明文口令与 TLS 私钥，
// 原样落进操作日志等于把凭据复制到另一张表（且日志保留期通常更长）。
// extra 可以保留——它只含档位/主机名/指纹这类可读项。
func (f InstanceForm) AuditReplica() map[string]any {
	replica := map[string]any{
		"id":                 f.Id,
		"name":               f.Name,
		"type":               f.Type,
		"host":               f.Host,
		"port":               f.Port,
		"params":             f.Params,
		"remark":             f.Remark,
		"sshTunnelMachineId": f.SshTunnelMachineId,
		"extra":              f.Extra,
		"tagCodePaths":       f.TagCodePaths,
	}
	if f.TlsMaterial != nil && !f.TlsMaterial.Empty() {
		replica["tlsMaterialSubmitted"] = true
	}
	// 凭证只记用户名与数量，不记密文
	replica["authCertUsers"] = collx.ArrayMap(f.AuthCerts, func(ac *tagentity.ResourceAuthCert) string {
		if ac == nil {
			return ""
		}
		return ac.Username
	})
	return replica
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
