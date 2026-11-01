package entity

import (
	"mayfly-go/pkg/model"
)

// DbColumnAcl 列级访问控制条目：把某张表的某一列对指定角色设为「禁止访问」。
//
// 与表级黑名单同一套语义（RoleIds 为空 = 所有账号），区别只是粒度到列。
// 列级不用「脱敏」表达：脱敏保留列、只改值，适合「看得到但不能看全」；
// 本表表达的是「这一列根本不该出现在结果里」——合规上通常对应手机号、证件号、
// 密钥这类字段，宁可查不出来，也不要给出一个看起来像真值的掩码。
type DbColumnAcl struct {
	model.Model

	DbId    uint64   `json:"dbId" gorm:"not null;index:idx_db_column_acl_db;comment:数据库id"`
	Table   string   `json:"tableName" gorm:"column:table_name;size:150;not null;comment:表名"`
	Column  string   `json:"columnName" gorm:"column:column_name;size:150;not null;comment:列名"`
	RoleIds []uint64 `json:"roleIds" gorm:"type:json;serializer:json;comment:生效角色id列表，空=所有账号"`
	Remark  string   `json:"remark" gorm:"size:255;comment:备注"`
}

func (DbColumnAcl) TableName() string {
	return "t_db_column_acl"
}
