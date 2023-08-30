package entity

import (
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Name       string `orm:"column(name)" json:"name"`
	Database   string `orm:"column(database)" json:"database"`
	Remark     string `json:"remark"`
	TagId      uint64
	TagPath    string
	InstanceId uint64
}
