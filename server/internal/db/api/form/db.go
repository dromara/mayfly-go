package form

type DbForm struct {
	Id         uint64 `json:"id"`
	Name       string `binding:"required" json:"name"`
	Database   string `json:"database"`
	Remark     string `json:"remark"`
	TagId      uint64 `binding:"required" json:"tagId"`
	TagPath    string `binding:"required" json:"tagPath"`
	InstanceId uint64 `binding:"required" json:"instanceId"`
}

type DbSqlSaveForm struct {
	Name string `json:"name" binding:"required"`
	Sql  string `json:"sql" binding:"required"`
	Type int    `json:"type" binding:"required"`
	Db   string `json:"db" binding:"required"`
}

// 数据库SQL执行表单
type DbSqlExecForm struct {
	Db     string `binding:"required" json:"db"`  //数据库名
	Sql    string `binding:"required" json:"sql"` // 执行sql
	Remark string `json:"remark"`                 // 执行备注
}
