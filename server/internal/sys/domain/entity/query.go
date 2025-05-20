package entity

import "mayfly-go/pkg/model"

type AccountQuery struct {
	model.PageParam
	Ids      []uint64 `json:"ids"`
	Name     string   `json:"name" form:"name"`
	Username string   `json:"code" form:"code"`
}

type SysLogQuery struct {
	model.PageParam
	CreatorId   uint64 `json:"creatorId" form:"creatorId"`
	Type        int8   `json:"type" form:"type"`
	Description string `json:"description" form:"description"`
}

type RoleQuery struct {
	model.PageParam
	Ids    []uint64 `json:"ids"`
	Name   string   `json:"name" form:"name"`
	Code   string   `json:"code" form:"code"`
	NotIds []uint64 `json:"notIds"`
}

type RoleAccountQuery struct {
	model.PageParam
	RoleId   uint64 `json:"roleId" `
	Name     string `json:"name" form:"name"`
	Username string `json:"username" form:"username"`
}
