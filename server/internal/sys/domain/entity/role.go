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
	Status int    `json:"status" gorm:"not null;"` // 1：可用；-1：不可用
	Name   string `json:"name" gorm:"size:32;not null;"`
	Remark string `json:"remark" gorm:"size:255;not null;"`
	Code   string `json:"code" gorm:"size:64;not null;"`
	Type   int8   `json:"type" gorm:"not null;comment:类型：1:公共角色；2:特殊角色;"`
}

func (a *Role) TableName() string {
	return "t_sys_role"
}

// 角色资源
type RoleResource struct {
	model.DeletedModel

	RoleId     uint64     `json:"roleId" gorm:"not null;"`
	ResourceId uint64     `json:"resourceId" gorm:"not null;"`
	CreateTime *time.Time `json:"createTime" gorm:"not null;"`
	CreatorId  uint64     `json:"creatorId" gorm:"not null;"`
	Creator    string     `json:"creator" gorm:"size:32;not null;"`
}

func (a *RoleResource) TableName() string {
	return "t_sys_role_resource"
}

// 账号角色
type AccountRole struct {
	model.DeletedModel

	AccountId  uint64     `json:"accountId" gorm:"not null;"`
	RoleId     uint64     `json:"roleId" gorm:"not null;"`
	CreateTime *time.Time `json:"createTime" gorm:"not null;"`
	CreatorId  uint64     `json:"creatorId" gorm:"not null;"`
	Creator    string     `json:"creator" gorm:"size:32;not null;"`
}

func (a *AccountRole) TableName() string {
	return "t_sys_account_role"
}
