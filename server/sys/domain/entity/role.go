package entity

import (
	"mayfly-go/base/model"
	"time"
)

type Role struct {
	model.Model
	Status int    `json:"status"` // 1：可用；-1：不可用
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

// 角色资源
type RoleResource struct {
	Id         uint64     `json:"id"`
	RoleId     uint64     `json:"roleId"`
	ResourceId uint64     `json:"resourceId"`
	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`
}

// 账号角色
type AccountRole struct {
	Id         uint64     `json:"id"`
	AccountId  uint64     `json:"accountId"`
	RoleId     uint64     `json:"roleId"`
	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`
}
