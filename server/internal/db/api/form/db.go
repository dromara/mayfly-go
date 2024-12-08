package form

import "mayfly-go/internal/db/domain/entity"

type DbForm struct {
	Id              uint64                   `json:"id"`
	Name            string                   `binding:"required" json:"name"`
	GetDatabaseMode entity.DbGetDatabaseMode `json:"getDatabaseMode"` // 获取数据库方式
	Database        string                   `json:"database"`
	Remark          string                   `json:"remark"`
	InstanceId      uint64                   `binding:"required" json:"instanceId"`
	AuthCertName    string                   `json:"authCertName"`
}

type DbSqlSaveForm struct {
	Name string `json:"name" binding:"required"`
	Sql  string `json:"sql" binding:"required"`
	Type int    `json:"type" binding:"required"`
	Db   string `json:"db" binding:"required"`
}

// 数据库SQL执行表单
type DbSqlExecForm struct {
	ExecId string `json:"execId"`                 // 执行id(用于取消执行使用)
	Db     string `binding:"required" json:"db"`  //数据库名
	Sql    string `binding:"required" json:"sql"` // 执行sql
	Remark string `json:"remark"`                 // 执行备注
}

// 数据库复制表
type DbCopyTableForm struct {
	Id        uint64 `binding:"required" json:"id"`
	Db        string `binding:"required" json:"db" `
	TableName string `binding:"required" json:"tableName"`
	CopyData  bool   `json:"copyData"` // 是否复制数据
}
