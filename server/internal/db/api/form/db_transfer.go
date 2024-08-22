package form

type DbTransferTaskForm struct {
	Id             uint64 `json:"id"`
	TaskName       string `binding:"required" json:"taskName"`       // 任务名称
	CheckedKeys    string `binding:"required" json:"checkedKeys"`    // 选中需要迁移的表
	DeleteTable    int    `binding:"required" json:"deleteTable"`    // 创建表前是否删除表 1是  2否
	NameCase       int    `binding:"required" json:"nameCase"`       // 表名、字段大小写转换  1无  2大写  3小写
	Strategy       int    `binding:"required" json:"strategy"`       // 迁移策略  1全量  2增量
	SrcDbId        int    `binding:"required" json:"srcDbId"`        // 源库id
	SrcDbName      string `binding:"required" json:"srcDbName"`      // 源库名
	SrcDbType      string `binding:"required" json:"srcDbType"`      // 源库类型
	SrcInstName    string `binding:"required" json:"srcInstName"`    // 源库实例名
	SrcTagPath     string `binding:"required" json:"srcTagPath"`     // 源库tagPath
	TargetDbId     int    `binding:"required" json:"targetDbId"`     // 目标库id
	TargetDbName   string `binding:"required" json:"targetDbName"`   // 目标库名
	TargetDbType   string `binding:"required" json:"targetDbType"`   // 目标库类型
	TargetInstName string `binding:"required" json:"targetInstName"` // 目标库实例名
	TargetTagPath  string `binding:"required" json:"targetTagPath"`  // 目标库tagPath
}
