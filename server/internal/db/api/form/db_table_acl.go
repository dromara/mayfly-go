package form

// DbTableAclForm 表访问控制表单。
//
// RoleIds 为空表示对所有账号生效（见 entity.DbTableAcl 注释），因此这里不能加 required：
// 「只保护表、不区分角色」是最常用的一种配置。
type DbTableAclForm struct {
	Id        uint64 `json:"id"`
	DbId      uint64 `binding:"required" json:"dbId"`
	TableName string `binding:"required" json:"tableName"`
	// Effect 1 禁止（默认）/ 2 允许；缺省按禁止处理
	Effect  int8     `json:"effect"`
	RoleIds []uint64 `json:"roleIds"`
	Remark  string   `json:"remark"`
}
