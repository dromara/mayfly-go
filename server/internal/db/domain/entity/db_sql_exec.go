package entity

import (
	"mayfly-go/pkg/model"
)

// 数据库sql执行记录
type DbSQLExec struct {
	model.Model

	DbId     uint64          `json:"dbId" gorm:"not null;"`
	Db       string          `json:"db" gorm:"size:150;not null;"`
	Table    string          `json:"table" gorm:"size:150;"`
	Type     DbSQLExecType   `json:"type" gorm:"not null;"`                     // 类型
	SQL      string          `json:"sql" gorm:"column:sql;size:5000;not null;"` // 执行的sql
	OldValue string          `json:"oldValue" gorm:"size:5000;"`
	Remark   string          `json:"remark" gorm:"size:255;"`
	Status   DbSQLExecStatus `json:"status"`                // 执行状态
	Res      string          `json:"res" gorm:"size:1000;"` // 执行结果

	FlowBizKey string `json:"flowBizKey" gorm:"size:50;index:idx_flow_biz_key;comment:流程关联的业务key"` // 流程业务key
}

// DbSQLExecType SQL 执行类型
type DbSQLExecType int8

const (
	DbSQLExecTypeOther  DbSQLExecType = -1 // 其他类型
	DbSQLExecTypeUpdate DbSQLExecType = 1  // 更新类型
	DbSQLExecTypeDelete DbSQLExecType = 2  // 删除类型
	DbSQLExecTypeInsert DbSQLExecType = 3  // 插入类型
	DbSQLExecTypeQuery  DbSQLExecType = 4  // 查询类型，如select、show等
	DbSQLExecTypeDDL    DbSQLExecType = 5  // DDL
)

// DbSQLExecStatus SQL 执行状态
type DbSQLExecStatus int8

const (
	DbSQLExecStatusWait    DbSQLExecStatus = 1
	DbSQLExecStatusSuccess DbSQLExecStatus = 2
	DbSQLExecStatusNo      DbSQLExecStatus = -1 // 不执行
	DbSQLExecStatusFail    DbSQLExecStatus = -2
)
