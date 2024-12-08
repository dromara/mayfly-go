package form

type DbTransferTaskForm struct {
	Id uint64 `json:"id"`

	TaskName         string `binding:"required" json:"taskName"` // 任务名称
	CronAble         int    `json:"cronAble"`                    // 是否定时  1是 -1否
	Cron             string `json:"cron"`                        // 定时任务cron表达式
	Mode             int    `binding:"required" json:"mode"`     // 数据迁移方式，1、迁移到数据库  2、迁移到文件
	TargetFileDbType string `json:"targetFileDbType"`            // 目标文件数据库类型
	FileSaveDays     int    `json:"fileSaveDays"`                // 文件保存天数
	Status           int    `json:"status" form:"status"`        // 启用状态 1启用 -1禁用

	CheckedKeys string `binding:"required" json:"checkedKeys"` // 选中需要迁移的表
	DeleteTable int    `binding:"required" json:"deleteTable"` // 创建表前是否删除表 1是  2否
	NameCase    int    `binding:"required" json:"nameCase"`    // 表名、字段大小写转换  1无  2大写  3小写
	Strategy    int    `binding:"required" json:"strategy"`    // 迁移策略  1全量  2增量

	SrcDbId     int    `binding:"required" json:"srcDbId"`     // 源库id
	SrcDbName   string `binding:"required" json:"srcDbName"`   // 源库名
	SrcDbType   string `binding:"required" json:"srcDbType"`   // 源库类型
	SrcInstName string `binding:"required" json:"srcInstName"` // 源库实例名
	SrcTagPath  string `binding:"required" json:"srcTagPath"`  // 源库tagPath

	TargetDbId     int    `json:"targetDbId"`     // 目标库id
	TargetDbName   string `json:"targetDbName"`   // 目标库名
	TargetDbType   string `json:"targetDbType"`   // 目标库类型
	TargetInstName string `json:"targetInstName"` // 目标库实例名
	TargetTagPath  string `json:"targetTagPath"`  // 目标库tagPath
}
type DbTransferTaskStatusForm struct {
	Id     uint64 `binding:"required" json:"taskId" form:"taskId"`
	Status int    `json:"status" form:"status"`
}
type DbTransferFileForm struct {
	Id       uint64 `json:"id"`
	FileName string `json:"fileName" form:"fileName"`
}
type DbTransferFileRunForm struct {
	Id           uint64 `json:"id"`                               // 文件ID
	TargetDbId   uint64 `json:"targetDbId" form:"targetDbId"`     // 需要执行sql的数据库id
	TargetDbName string `json:"targetDbName" form:"targetDbName"` // 需要执行sql的数据库名
	ClientId     string `json:"clientId" form:"clientId"`         // 客户端的唯一id，用于消息回传
}
