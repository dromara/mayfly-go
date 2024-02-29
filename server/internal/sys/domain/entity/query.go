package entity

type AccountQuery struct {
	Ids      []uint64 `json:"ids"`
	Name     string   `json:"name" form:"name"`
	Username string   `json:"code" form:"code"`
}

type SysLogQuery struct {
	CreatorId   uint64 `json:"creatorId" form:"creatorId"`
	Type        int8   `json:"type" form:"type"`
	Description string `json:"description" form:"description"`
}

type RoleQuery struct {
	Ids    []uint64 `json:"ids"`
	Name   string   `json:"name" form:"name"`
	Code   string   `json:"code" form:"code"`
	NotIds []uint64 `json:"notIds"`
}

type RoleAccountQuery struct {
	RoleId   uint64 `json:"roleId" `
	Name     string `json:"name" form:"name"`
	Username string `json:"username" form:"username"`
}
