package vo

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/model"
	"time"
)

type AccountManageVO struct {
	model.Model
	model.ExtraData

	Name          string               `json:"name"`
	Username      string               `json:"username"`
	Mobile        string               `json:"mobile"`
	Email         string               `json:"email"`
	Status        entity.AccountStatus `json:"status"`
	LastLoginTime *time.Time           `json:"lastLoginTime"`
	OtpSecret     string               `json:"otpSecret"`
}

type SimpleAccountVO struct {
	Id       uint64 `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email"`

	Roles []*AccountRoleVO `json:"roles" gorm:"-"`
}

// 账号角色信息
type AccountRoleVO struct {
	RoleId        uint64               `json:"roleId"`
	RoleName      string               `json:"roleName"`
	Code          string               `json:"code"`
	Status        int                  `json:"status"`
	AccountId     uint64               `json:"accountId" gorm:"column:accountId"`
	AccountName   string               `json:"accountName" gorm:"column:accountName"`
	Username      string               `json:"username"`
	AccountStatus entity.AccountStatus `json:"accountStatus" gorm:"column:accountStatus"`
	CreateTime    *time.Time           `json:"createTime"`
	Creator       string               `json:"creator"`
}

// 账号个人信息
type AccountPersonVO struct {
	Roles []*AccountRoleVO `json:"roles"` // 角色信息
}
