package entity

import (
	"mayfly-go/base/model"
)

type Account struct {
	model.Model

	Username string `json:"username"`
	Password string `json:"-"`
	Status   int8   `json:"status"`
}

// 是否可用
func (a *Account) IsEnable() bool {
	return a.Status == AccountEnableStatus
}

const (
	AccountEnableStatus  int8 = 1  // 启用状态
	AccountDisableStatus int8 = -1 // 禁用状态
)
