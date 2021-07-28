package vo

import (
	"mayfly-go/base/model"
	"time"
)

type AccountManageVO struct {
	model.Model
	Username      *string    `json:"username"`
	Status        int        `json:"status"`
	LastLoginTime *time.Time `json:"lastLoginTime"`
}

type AccountRoleVO struct {
	Name       *string    `json:"name"`
	Status     int        `json:"status"`
	CreateTime *time.Time `json:"createTime"`
	Creator    string     `json:"creator"`
}
