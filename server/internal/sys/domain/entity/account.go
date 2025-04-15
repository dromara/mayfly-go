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
	model.ExtraData

	Name          string        `json:"name" gorm:"size:30;not null;"`
	Username      string        `json:"username" gorm:"size:30;not null;"`
	Mobile        string        `json:"mobile" gorm:"size:20;"`
	Email         string        `json:"email" gorm:"size:100;"`
	Password      string        `json:"-" gorm:"size:64;not null;"`
	Status        AccountStatus `json:"status" gorm:"not null;"`
	LastLoginTime *time.Time    `json:"lastLoginTime"`
	LastLoginIp   string        `json:"lastLoginIp" gorm:"size:50;"`
	OtpSecret     string        `json:"-" gorm:"size:100;"`
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
