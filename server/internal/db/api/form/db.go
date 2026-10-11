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

type DbSQLSaveForm struct {
	Name string `json:"name" binding:"required"`
	SQL  string `json:"sql" binding:"required"`
	Type int    `json:"type" binding:"required"`
	Db   string `json:"db" binding:"required"`
}

// 数据库SQL执行表单
type DbSQLExecForm struct {
	ExecId string `json:"execId"`                 // 执行id(用于取消执行使用)
	Db     string `binding:"required" json:"db"`  //数据库名
	SQL    string `binding:"required" json:"sql"` // 执行sql
	Remark string `json:"remark"`                 // 执行备注
	// AckWarn 操作者已在确认框里选择「直接执行」，用于命中「仅提醒」后的重试
	AckWarn bool `json:"ackWarn"`
	// AskWarn 该次执行能否弹确认：单条执行可以，批量选区不能逐条追问
	AskWarn bool `json:"askWarn"`
	// Atomic 整批原子执行：全部成功才提交，任一条失败即整批回滚
	Atomic bool `json:"atomic"`
}

// DbSqlAnalysisForm SQL 只读分析入参：供取执行计划入口绑定（sql 原文与已切割的 sqls 合并成一批语句）
type DbSqlAnalysisForm struct {
	Id   uint64   `json:"id" binding:"required"` // 数据库id
	Db   string   `json:"db" binding:"required"` // 数据库名
	SQLs []string `json:"sqls"`                  // 待分析语句（已按方言切割）
	SQL  string   `json:"sql"`                   // 单条待分析语句，与 SQLs 二选一
}

// QueryExportForm 查询结果集全量导出入参
type QueryExportForm struct {
	Db     string `json:"db" binding:"required"`  // 数据库名
	SQL    string `json:"sql" binding:"required"` // 要导出的查询语句（单条）
	Format string `json:"format"`                 // 导出格式（csv/json，默认 csv）
}

// DbSchemaDiffForm 结构对比（库间表结构漂移检测）入参
type DbSchemaDiffForm struct {
	SrcDbId uint64   `json:"srcDbId" binding:"required"` // 源库id
	SrcDb   string   `json:"srcDb" binding:"required"`   // 源库名
	DstDbId uint64   `json:"dstDbId" binding:"required"` // 目标库id
	DstDb   string   `json:"dstDb" binding:"required"`   // 目标库名
	Tables  []string `json:"tables"`                     // 限定比对的表名；为空则比对两侧表名并集（限额内）
}

// DbSchemaBaselineForm 表结构基线入参（建立/刷新与删除共用）
type DbSchemaBaselineForm struct {
	Id     uint64   `json:"id" binding:"required"` // 数据库id
	Db     string   `json:"db" binding:"required"` // 库名
	Tables []string `json:"tables"`                // 限定纳入基线的表名；为空则该库全部表（限额内）
}

// DbSessionCancelForm 取消数据库会话
type DbSessionCancelForm struct {
	Id        uint64 `json:"id" binding:"required"`        // 数据库id
	Db        string `json:"db" binding:"required"`        // 数据库名
	SessionId string `json:"sessionId" binding:"required"` // 会话标识（来自活动会话列表）
}

// DbCancelExecForm 取消一次进行中的执行
type DbCancelExecForm struct {
	ExecId string `json:"execId" binding:"required"`
}

// 数据库复制表
type DbCopyTableForm struct {
	Id        uint64 `binding:"required" json:"id"`
	Db        string `binding:"required" json:"db" `
	TableName string `binding:"required" json:"tableName"`
	CopyData  bool   `json:"copyData"` // 是否复制数据
}
