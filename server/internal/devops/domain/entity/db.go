package entity

import (
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Name      string `orm:"column(name)" json:"name"`
	Type      string `orm:"column(type)" json:"type"` // 类型，mysql oracle等
	Host      string `orm:"column(host)" json:"host"`
	Port      int    `orm:"column(port)" json:"port"`
	Network   string `orm:"column(network)" json:"network"`
	Username  string `orm:"column(username)" json:"username"`
	Password  string `orm:"column(password)" json:"-"`
	Database  string `orm:"column(database)" json:"database"`
	Params    string `json:"params"`
	ProjectId uint64
	Project   string
	EnvId     uint64
	Env       string
}
