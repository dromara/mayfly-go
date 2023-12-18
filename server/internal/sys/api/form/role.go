package form

// 分配角色资源表单信息
type RoleResourceForm struct {
	Id          uint64 `json:"id"`
	ResourceIds string `json:"resourceIds"`
}

// 保存角色信息表单
type RoleForm struct {
	Id     int    `json:"id"`
	Status int    `json:"status"` // 1：可用；-1：不可用
	Name   string `json:"name" binding:"required"`
	Code   string `json:"code" binding:"required"`
	Remark string `json:"remark"`
}

// 账号分配角色表单
type AccountRoleForm struct {
	Id         uint64 `json:"id" binding:"required"`
	RoleId     uint64 `json:"roleId" binding:"required"`
	RelateType int    `json:"relateType" binding:"required"`
}
