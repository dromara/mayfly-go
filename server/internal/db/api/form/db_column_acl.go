package form

// DbColumnAclForm 列级访问控制表单。RoleIds 为空 = 所有账号（与表级同语义）。
type DbColumnAclForm struct {
	Id         uint64   `json:"id"`
	DbId       uint64   `binding:"required" json:"dbId"`
	TableName  string   `binding:"required" json:"tableName"`
	ColumnName string   `binding:"required" json:"columnName"`
	RoleIds    []uint64 `json:"roleIds"`
	Remark     string   `json:"remark"`
}
