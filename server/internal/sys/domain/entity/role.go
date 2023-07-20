package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

const (
	RoleTypeCommon  int = 1 // 公共角色类型
	RoleTypeSpecial int = 2 // 特殊角色类型
)

type Role struct {
	model.Model
	Status int    `json:"status"` // 1：可用；-1：不可用
	Name   string `json:"name"`
	Remark string `json:"remark"`
	Code   string `json:"code"`
	Type   int    `json:"type"`
}

func (a *Role) TableName() string {
	return "t_sys_role"
}

// 角色资源
type RoleResource struct {
	model.DeletedModel

	RoleId     uint64     `json:"roleId"`
	ResourceId uint64     `json:"resourceId"`
	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`
}

func (a *RoleResource) TableName() string {
	return "t_sys_role_resource"
}

// 账号角色
type AccountRole struct {
	model.DeletedModel

	AccountId  uint64     `json:"accountId"`
	RoleId     uint64     `json:"roleId"`
	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`
}

func (a *AccountRole) TableName() string {
	return "t_sys_account_role"
}
