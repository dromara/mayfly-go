package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type Account struct {
	model.Model

	Name          string     `json:"name"`
	Username      string     `json:"username"`
	Password      string     `json:"-"`
	Status        int8       `json:"status"`
	LastLoginTime *time.Time `json:"lastLoginTime"`
	LastLoginIp   string     `json:"lastLoginIp"`
	OtpSecret     string     `json:"-"`
}

func (a *Account) TableName() string {
	return "t_sys_account"
}

// 是否可用
func (a *Account) IsEnable() bool {
	return a.Status == AccountEnableStatus
}

const (
	AccountEnableStatus  int8 = 1  // 启用状态
	AccountDisableStatus int8 = -1 // 禁用状态
)
