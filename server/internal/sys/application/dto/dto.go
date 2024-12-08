package dto

import "time"

type ResourceRole struct {
	RoleId       uint64     `json:"roleId"`
	RoleName     string     `json:"roleName"`
	RoleCode     string     `json:"roleCode"`
	RoleStatus   int        `json:"roleStatus"`   // 角色状态
	AllocateTime *time.Time `json:"allocateTime"` // 分配时间
	Assigner     string     `json:"assigner"`     // 分配人
}
