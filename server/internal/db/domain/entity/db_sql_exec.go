package entity

import (
	"mayfly-go/pkg/model"
)

// 数据库sql执行记录
type DbSqlExec struct {
	model.Model `orm:"-"`

	DbId     uint64 `json:"dbId" gorm:"not null;"`
	Db       string `json:"db" gorm:"size:150;not null;"`
	Table    string `json:"table" gorm:"size:150;"`
	Type     int8   `json:"type" gorm:"not null;"`          // 类型
	Sql      string `json:"sql" gorm:"size:5000;not null;"` // 执行的sql
	OldValue string `json:"oldValue" gorm:"size:5000;"`
	Remark   string `json:"remark" gorm:"size:255;"`
	Status   int8   `json:"status"`                // 执行状态
	Res      string `json:"res" gorm:"size:1000;"` // 执行结果

	FlowBizKey string `json:"flowBizKey" gorm:"size:50;index:idx_flow_biz_key;comment:流程关联的业务key"` // 流程业务key
}

const (
	DbSqlExecTypeOther  int8 = -1 // 其他类型
	DbSqlExecTypeUpdate int8 = 1  // 更新类型
	DbSqlExecTypeDelete int8 = 2  // 删除类型
	DbSqlExecTypeInsert int8 = 3  // 插入类型
	DbSqlExecTypeQuery  int8 = 4  // 查询类型，如select、show等
	DbSqlExecTypeDDL    int8 = 5  // DDL

	DbSqlExecStatusWait    = 1
	DbSqlExecStatusSuccess = 2
	DbSqlExecStatusNo      = -1 // 不执行
	DbSqlExecStatusFail    = -2
)
