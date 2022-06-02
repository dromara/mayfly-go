package form

// 分配角色资源表单信息
type RoleResourceForm struct {
	Id          int
	ResourceIds string
}

// 保存角色信息表单
type RoleForm struct {
	Id     int
	Status int    `json:"status"` // 1：可用；-1：不可用
	Name   string `binding:"required"`
	Code   string `binding:"required"`
	Remark string `json:"remark"`
}

// 账号分配角色表单
type AccountRoleForm struct {
	Id      int `binding:"required"`
	RoleIds string
}
