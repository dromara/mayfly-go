package entity

import "mayfly-go/pkg/model"

// 数据库实例查询
type InstanceQuery struct {
	model.Model

	Name     string `orm:"column(name)" json:"name" form:"name"`
	Type     string `orm:"column(type)" json:"type"` // 类型，mysql oracle等
	Host     string `orm:"column(host)" json:"host"`
	Port     int    `orm:"column(port)" json:"port"`
	Network  string `orm:"column(network)" json:"network"`
	Username string `orm:"column(username)" json:"username"`
	Password string `orm:"column(password)" json:"-"`
	Params   string `orm:"column(params)" json:"params"`
	Remark   string `orm:"column(remark)" json:"remark"`
}

// 数据库查询实体，不与数据库表字段一一对应
type DbQuery struct {
	model.Model

	Name     string `orm:"column(name)" json:"name"`
	Type     string `orm:"column(type)" json:"type"` // 类型，mysql oracle等
	Host     string `orm:"column(host)" json:"host"`
	Port     int    `orm:"column(port)" json:"port"`
	Network  string `orm:"column(network)" json:"network"`
	Username string `orm:"column(username)" json:"username"`
	Password string `orm:"column(password)" json:"-"`
	Database string `orm:"column(database)" json:"database"`
	Params   string `json:"params"`
	Remark   string `json:"remark"`

	TagIds  []uint64
	TagPath string `form:"tagPath"`
}

type DbSqlExecQuery struct {
	Id    uint64 `json:"id" form:"id"`
	DbId  uint64 `json:"dbId" form:"dbId"`
	Db    string `json:"db" form:"db"`
	Table string `json:"table" form:"table"`
	Type  int8   `json:"type" form:"type"` // 类型

	CreatorId uint64
}
