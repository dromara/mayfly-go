package entity

import (
	"mayfly-go/pkg/model"
)

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
	Status   int8   `json:"status"` // 执行状态
	Res      string `json:"res"`    // 执行结果

	FlowBizKey string `json:"flowBizKey"` // 流程业务key
}

const (
	DbSqlExecTypeOther  int8 = -1 // 其他类型
	DbSqlExecTypeUpdate int8 = 1  // 更新类型
	DbSqlExecTypeDelete int8 = 2  // 删除类型
	DbSqlExecTypeInsert int8 = 3  // 插入类型
	DbSqlExecTypeQuery  int8 = 4  // 查询类型，如select、show等

	DbSqlExecStatusWait    = 1
	DbSqlExecStatusSuccess = 2
	DbSqlExecStatusNo      = -1 // 不执行
	DbSqlExecStatusFail    = -2
)
