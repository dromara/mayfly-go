package entity

import (
	"mayfly-go/pkg/model"
)

type Redis struct {
	model.Model

	Host      string `orm:"column(host)" json:"host"`
	Password  string `orm:"column(password)" json:"-"`
	Db        int    `orm:"column(database)" json:"db"`
	ProjectId uint64
	Project   string
	EnvId     uint64
	Env       string
}
