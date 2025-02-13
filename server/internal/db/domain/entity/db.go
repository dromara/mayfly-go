package entity

import (
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Code            string            `json:"code" gorm:"size:32;not null;index:idx_code"`
	Name            string            `json:"name" gorm:"size:255;not null;"`
	GetDatabaseMode DbGetDatabaseMode `json:"getDatabaseMode" gorm:"comment:库名获取方式（-1.实时获取、1.指定库名）"` // 获取数据库方式
	Database        string            `json:"database" gorm:"size:2000;"`
	Remark          string            `json:"remark" gorm:"size:255;"`
	InstanceId      uint64            `json:"instanceId" gorm:"not null;"`
	AuthCertName    string            `json:"authCertName" gorm:"size:255;"`
}

type DbGetDatabaseMode int8

const (
	DbGetDatabaseModeAuto   DbGetDatabaseMode = -1 // 自动获取（根据凭证获取对应所有库名）
	DbGetDatabaseModeAssign DbGetDatabaseMode = 1  // 指定库名
)
