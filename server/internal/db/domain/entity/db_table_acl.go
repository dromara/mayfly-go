package entity

import (
	"mayfly-go/pkg/model"
)

// DbTableAcl 表访问控制条目：把某张表对指定角色设为「禁止访问」。
//
// 采用黑名单语义而不是白名单：白名单会让「以后新建的表」默认对所有人不可见，
// 存量部署升级后等于大面积断供；保护敏感表只需显式列出要挡住的那几张。
//
// 判定按表名（忽略大小写）与角色求交集：RoleIds 为空表示对所有账号生效——用空集
// 表达「所有人」，而不是塞一个哨兵角色 id（角色被删后哨兵会失去语义）。
type DbTableAcl struct {
	model.Model

	DbId uint64 `json:"dbId" gorm:"not null;index:idx_db_table_acl_db;comment:数据库id"`
	// 字段名用 Table 而不是 TableName：后者与 TableName() 方法同名，Go 不允许
	Table string `json:"tableName" gorm:"column:table_name;size:150;not null;comment:表名"`
	// Effect 规则类型：1 禁止（默认，黑名单）/ 2 允许（白名单）。
	//
	// 白名单一旦出现在某个库里，该库就切到「默认拒绝」模式：不在允许清单里的表对所有账号不可见。
	// 默认值取 1 而不是 0：零值在 GORM 更新里会被跳过，也让「老数据没有这一列」天然按黑名单解释
	Effect  int8     `json:"effect" gorm:"not null;default:1;comment:1禁止 2允许"`
	RoleIds []uint64 `json:"roleIds" gorm:"type:json;serializer:json;comment:生效角色id列表，空=所有账号"`
	Remark  string   `json:"remark" gorm:"size:255;comment:备注"`
}

// 表访问控制规则类型
const (
	// DbTableAclEffectDeny 黑名单：命中的账号不能访问该表
	DbTableAclEffectDeny int8 = 1
	// DbTableAclEffectAllow 白名单：该库存在任何一条 allow 规则时切到默认拒绝模式
	DbTableAclEffectAllow int8 = 2
)

func (DbTableAcl) TableName() string {
	return "t_db_table_acl"
}
