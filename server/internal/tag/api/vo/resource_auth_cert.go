package vo

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"time"
)

type ResourceAuthCert struct {
	model.ExtraData

	Id             uint64                        `json:"id"`
	Name           string                        `json:"name"`           // 名称
	ResourceCode   string                        `json:"resourceCode"`   // 资源编号
	ResourceType   int8                          `json:"resourceType"`   // 资源类型
	Username       string                        `json:"username"`       // 用户名
	Ciphertext     string                        `json:"ciphertext"`     // 密文
	CiphertextType entity.AuthCertCiphertextType `json:"ciphertextType"` // 密文类型
	Type           entity.AuthCertType           `json:"type"`           // 凭证类型
	Remark         string                        `json:"remark"`         // 备注

	CreateTime *time.Time `json:"createTime"`
}

// 授权凭证基础信息
type AuthCertBaseVO struct {
	Id   int    `json:"id"`
	Name string `json:"name"` // 名称（全局唯一）

	ResourceCode   string                        `json:"resourceCode"`   // 资源编号
	ResourceType   int8                          `json:"resourceType"`   // 资源类型
	Type           entity.AuthCertType           `json:"type"`           // 凭证类型
	Username       string                        `json:"username"`       // 用户名
	CiphertextType entity.AuthCertCiphertextType `json:"ciphertextType"` // 密文类型
	Remark         string                        `json:"remark"`         // 备注
}

type SimpleResourceAuthCert struct {
	Id       uint64 `json:"id"`
	Name     string `json:"code"`     // 名称
	Username string `json:"username"` // 用户名
}
