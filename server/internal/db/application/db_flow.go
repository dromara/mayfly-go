package application

import flowapp "mayfly-go/internal/flow/application"

const (
	DbSqlExecFlowBizType = "db_sql_exec_flow" // db sql exec flow biz type
)

func InitDbFlowHandler() {
	flowapp.RegisterBizHandler(DbSqlExecFlowBizType, GetDbSqlExecApp())
}
