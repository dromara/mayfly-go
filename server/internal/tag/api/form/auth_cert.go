package form

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
)

// 授权凭证
type AuthCertForm struct {
	Id             uint64                        `json:"id"`
	Name           string                        `json:"name"`                              // 名称
	ResourceCode   string                        `json:"resourceCode"`                      // 资源编号
	ResourceType   int8                          `json:"resourceType"`                      // 资源类型
	Username       string                        `json:"username"`                          // 用户名
	Ciphertext     string                        `json:"ciphertext"`                        // 密文
	CiphertextType entity.AuthCertCiphertextType `json:"ciphertextType" binding:"required"` // 密文类型
	Extra          model.Map[string, any]        `json:"extra"`                             // 账号需要的其他额外信息（如秘钥口令等）
	Type           entity.AuthCertType           `json:"type" binding:"required"`           // 凭证类型
	Remark         string                        `json:"remark"`                            // 备注
}
