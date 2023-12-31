package entity

import (
	"errors"
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

func (ac *AuthCert) TableName() string {
	return "t_auth_cert"
}

const (
	AuthCertAuthMethodPassword int8 = 1 // 密码
	MachineAuthMethodPublicKey int8 = 2 // 密钥

	AuthCertTypePrivate int8 = 1
	AuthCertTypePublic  int8 = 2
)

// PwdEncrypt 密码加密
func (ac *AuthCert) PwdEncrypt() error {
	password, err := utils.PwdAesEncrypt(ac.Password)
	if err != nil {
		return errors.New("加密授权凭证密码失败")
	}
	passphrase, err := utils.PwdAesEncrypt(ac.Passphrase)
	if err != nil {
		return errors.New("加密授权凭证私钥失败")
	}
	ac.Password = password
	ac.Passphrase = passphrase
	return nil
}

// PwdDecrypt 密码解密
func (ac *AuthCert) PwdDecrypt() error {
	password, err := utils.PwdAesDecrypt(ac.Password)
	if err != nil {
		return errors.New("解密授权凭证密码失败")
	}
	passphrase, err := utils.PwdAesDecrypt(ac.Passphrase)
	if err != nil {
		return errors.New("解密授权凭证私钥失败")
	}
	ac.Password = password
	ac.Passphrase = passphrase
	return nil
}
