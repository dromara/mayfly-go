package entity

import (
	"errors"
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/model"
	"time"
)

type Account struct {
	model.Model

	Name          string        `json:"name"`
	Username      string        `json:"username"`
	Password      string        `json:"-"`
	Status        AccountStatus `json:"status"`
	LastLoginTime *time.Time    `json:"lastLoginTime"`
	LastLoginIp   string        `json:"lastLoginIp"`
	OtpSecret     string        `json:"-"`
}

func (a *Account) TableName() string {
	return "t_sys_account"
}

// 是否可用
func (a *Account) IsEnable() bool {
	return a.Status == AccountEnable
}

func (a *Account) OtpSecretEncrypt() error {
	secret, err := utils.PwdAesEncrypt(a.OtpSecret)
	if err != nil {
		return errors.New("加密账户密码失败")
	}
	a.OtpSecret = secret
	return nil
}

func (a *Account) OtpSecretDecrypt() error {
	if a.OtpSecret == "-" {
		return nil
	}
	secret, err := utils.PwdAesDecrypt(a.OtpSecret)
	if err != nil {
		return errors.New("解密账户密码失败")
	}
	a.OtpSecret = secret
	return nil
}

type AccountStatus int8

const (
	AccountEnable  AccountStatus = 1  // 启用状态
	AccountDisable AccountStatus = -1 // 禁用状态
)

var AccountStatusEnum = enumx.NewEnum[AccountStatus]("账号状态").
	Add(AccountEnable, "启用").
	Add(AccountDisable, "禁用")
