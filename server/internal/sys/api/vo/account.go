package vo

import (
	"mayfly-go/pkg/model"
	"time"
)

type AccountManageVO struct {
	model.Model
	Name          string     `json:"name"`
	Username      string     `json:"username"`
	Status        int        `json:"status"`
	LastLoginTime *time.Time `json:"lastLoginTime"`
	OtpSecret     string     `json:"otpSecret"`
}

// 账号角色信息
type AccountRoleVO struct {
	Name       *string    `json:"name"`
	Status     int        `json:"status"`
	CreateTime *time.Time `json:"createTime"`
	Creator    string     `json:"creator"`
}

// 账号个人信息
type AccountPersonVO struct {
	Roles []AccountRoleVO `json:"roles"` // 角色信息
}
