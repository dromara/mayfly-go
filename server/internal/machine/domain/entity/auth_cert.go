package entity

import (
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/model"
)

// 授权凭证
type AuthCert struct {
	model.Model

	Name       string `json:"name"`
	AuthMethod int8   `json:"authMethod"`                                         // 1.密码 2.秘钥
	Password   string `json:"password" gorm:"column:password;type:varchar(4200)"` // 密码or私钥
	Passphrase string `json:"passphrase"`                                         // 私钥口令
	Remark     string `json:"remark"`
}

func (a *AuthCert) TableName() string {
	return "t_auth_cert"
}

const (
	AuthCertAuthMethodPassword int8 = 1 // 密码
	MachineAuthMethodPublicKey int8 = 2 // 密钥

	AuthCertTypePrivate int8 = 1
	AuthCertTypePublic  int8 = 2
)

// 密码加密
func (ac *AuthCert) PwdEncrypt() {
	ac.Password = utils.PwdAesEncrypt(ac.Password)
	ac.Passphrase = utils.PwdAesEncrypt(ac.Passphrase)
}

// 密码解密
func (ac *AuthCert) PwdDecrypt() {
	ac.Password = utils.PwdAesDecrypt(ac.Password)
	ac.Passphrase = utils.PwdAesDecrypt(ac.Passphrase)
}
