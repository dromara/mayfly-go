package form

type DataSyncTaskForm struct {
	Id       uint64 `json:"id"`
	TaskName string `binding:"required" json:"taskName"`
	TaskCron string `binding:"required" json:"taskCron"`
	TaskKey  string `json:"taskKey"`
	Status   int    `binding:"required" json:"status"`

	SrcDbId     int64  `binding:"required" json:"srcDbId"`
	SrcDbName   string `binding:"required" json:"srcDbName"`
	SrcTagPath  string `binding:"required" json:"srcTagPath"`
	DataSql     string `binding:"required" json:"dataSql"`
	PageSize    int    `binding:"required" json:"pageSize"`
	UpdField    string `binding:"required" json:"updField"`
	UpdFieldVal string `binding:"required" json:"updFieldVal"`

	TargetDbId      int64  `binding:"required" json:"targetDbId"`
	TargetDbName    string `binding:"required" json:"targetDbName"`
	TargetTagPath   string `binding:"required" json:"targetTagPath"`
	TargetTableName string `binding:"required" json:"targetTableName"`
	FieldMap        string `binding:"required" json:"fieldMap"`
}

type DataSyncTaskStatusForm struct {
	Id     uint64 `binding:"required" json:"taskId"`
	Status int    `json:"status"`
}
