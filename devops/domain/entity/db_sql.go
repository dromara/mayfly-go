package entity

import (
	"mayfly-go/base/model"
)

type DbSql struct {
	model.Model `orm:"-"`

	DbId uint64 `json:"db_id"`
	Type int    `json:"type"` // 类型
	Sql  string `json:"sql"`
}
