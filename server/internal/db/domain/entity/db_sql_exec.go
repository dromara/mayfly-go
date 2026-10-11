package entity

import (
	"mayfly-go/pkg/model"
)

// 数据库sql执行记录
type DbSQLExec struct {
	model.Model

	DbId  uint64        `json:"dbId" gorm:"not null;"`
	Db    string        `json:"db" gorm:"size:150;not null;"`
	Table string        `json:"table" gorm:"size:150;"`
	Type  DbSQLExecType `json:"type" gorm:"not null;"`                     // 类型
	SQL   string        `json:"sql" gorm:"column:sql;size:5000;not null;"` // 执行的sql
	// OldValue 执行前采集的行级旧值（UPDATE=变更列+主键，DELETE=整行），回滚与审计的数据源。
	// 用 mediumtext 而非 varchar：上限 200 行的行级 JSON 轻易超过 varchar(5000)，
	// 超限时记录整体写不进去——连审计都会一起丢，而回滚能力本身也依赖它
	OldValue string `json:"oldValue" gorm:"type:mediumtext;"`
	// NewValue 执行后采集的行级快照（UPDATE=变更后状态；INSERT=插入的字面量值）。
	// 生成回滚时以它做并发校验与 CAS 条件：SQL 文本推不出真实写入值（表达式、函数、触发器、类型归一），
	// 只有执行后的实际状态能证明「这行至今没被再改过」
	NewValue string          `json:"newValue" gorm:"type:mediumtext;"`
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
