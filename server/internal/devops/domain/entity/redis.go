package entity

import (
	"mayfly-go/pkg/model"
)

type Redis struct {
	model.Model

	Host      string `orm:"column(host)" json:"host"`
	Mode      string `json:"mode"`
	Password  string `orm:"column(password)" json:"-"`
	Db        int    `orm:"column(database)" json:"db"`
	Remark    string
	ProjectId uint64
	Project   string
	EnvId     uint64
	Env       string
}

const (
	RedisModeStandalone = "standalone"
	RedisModeCluster    = "cluster"
)
