package entity

import (
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Code            string            `orm:"column(code)" json:"code"`
	Name            string            `orm:"column(name)" json:"name"`
	GetDatabaseMode DbGetDatabaseMode `json:"getDatabaseMode"` // 获取数据库方式
	Database        string            `orm:"column(database)" json:"database"`
	Remark          string            `json:"remark"`
	InstanceId      uint64
	AuthCertName    string `json:"authCertName"`
}

type DbGetDatabaseMode int8

const (
	DbGetDatabaseModeAuto   DbGetDatabaseMode = -1 // 自动获取（根据凭证获取对应所有库名）
	DbGetDatabaseModeAssign DbGetDatabaseMode = 1  // 指定库名
)
