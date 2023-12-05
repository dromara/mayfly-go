package entity

import (
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Code       string `orm:"column(code)" json:"code"`
	Name       string `orm:"column(name)" json:"name"`
	Database   string `orm:"column(database)" json:"database"`
	Remark     string `json:"remark"`
	InstanceId uint64
}
