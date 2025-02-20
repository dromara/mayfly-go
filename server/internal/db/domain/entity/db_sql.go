package entity

import (
	"mayfly-go/pkg/model"
)

// DbSql 用户保存的数据库sql
type DbSql struct {
	model.Model `orm:"-"`

	DbId uint64 `json:"dbId" gorm:"not null;"`
	Db   string `json:"db" gorm:"size:100;not null;"`
	Type int    `json:"type" gorm:"not null;"` // 类型
	Sql  string `json:"sql" gorm:"type:text;comment:sql语句"`
	Name string `json:"name" gorm:"size:255;not null;comment:sql模板名"`
}
