package entity

import "mayfly-go/pkg/model"

// 数据库sql执行记录
type DbSqlExec struct {
	model.Model `orm:"-"`

	DbId     uint64 `json:"dbId"`
	Db       string `json:"db"`
	Table    string `json:"table"`
	Type     int8   `json:"type"` // 类型
	Sql      string `json:"sql"`  // 执行的sql
	OldValue string `json:"oldValue"`
	Remark   string `json:"remark"`
}

const (
	DbSqlExecTypeUpdate int8 = 1 // 更新类型
	DbSqlExecTypeDelete int8 = 2 // 删除类型
	DbSqlExecTypeInsert int8 = 3 // 插入类型
)
