package form

type DbForm struct {
	Id                 uint64 `json:"id"`
	Name               string `binding:"required" json:"name"`
	Type               string `binding:"required" json:"type"` // 类型，mysql oracle等
	Host               string `binding:"required" json:"host"`
	Port               int    `binding:"required" json:"port"`
	Username           string `binding:"required" json:"username"`
	Password           string `json:"password"`
	Params             string `json:"params"`
	Database           string `json:"database"`
	Remark             string `json:"remark"`
	TagId              uint64 `json:"tagId"`
	TagPath            string `json:"tagPath"`
	SshTunnelMachineId int    `json:"sshTunnelMachineId"`
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
