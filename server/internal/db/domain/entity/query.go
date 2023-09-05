package entity

import "mayfly-go/pkg/model"

// 数据库实例查询
type InstanceQuery struct {
	Id   uint64 `json:"id" form:"id"`
	Name string `json:"name" form:"name"`
	Host string `json:"host" form:"host"`
}

// 数据库查询实体，不与数据库表字段一一对应
type DbQuery struct {
	model.Model

	Name     string `orm:"column(name)" json:"name"`
	Database string `orm:"column(database)" json:"database"`
	Remark   string `json:"remark"`

	TagIds  []uint64 `orm:"column(tag_id)"`
	TagPath string   `form:"tagPath"`

	InstanceId uint64 `form:"instanceId"`
}

type DbSqlExecQuery struct {
	Id    uint64 `json:"id" form:"id"`
	DbId  uint64 `json:"dbId" form:"dbId"`
	Db    string `json:"db" form:"db"`
	Table string `json:"table" form:"table"`
	Type  int8   `json:"type" form:"type"` // 类型

	CreatorId uint64
}
