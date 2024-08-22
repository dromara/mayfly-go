package vo

import "time"

type DbTransferTaskListVO struct {
	Id         *int64     `json:"id"`
	CreateTime *time.Time `json:"createTime"`
	Creator    string     `json:"creator"`
	UpdateTime *time.Time `json:"updateTime"`
	Modifier   string     `json:"modifier"`

	RunningState int    `json:"runningState"`
	LogId        uint64 `json:"logId"`
	TaskName     string `json:"taskName"` // 任务名称

	CheckedKeys string `json:"checkedKeys"` // 选中需要迁移的表
	DeleteTable int    `json:"deleteTable"` // 创建表前是否删除表
	NameCase    int    `json:"nameCase"`    // 表名、字段大小写转换  1无  2大写  3小写
	Strategy    int    `json:"strategy"`    // 迁移策略  1全量  2增量

	SrcDbId     int64  `json:"srcDbId"`     // 源库id
	SrcDbName   string `json:"srcDbName"`   // 源库名
	SrcTagPath  string `json:"srcTagPath"`  // 源库tagPath
	SrcDbType   string `json:"srcDbType"`   // 源库类型
	SrcInstName string `json:"srcInstName"` // 源库实例名

	TargetDbId     int    `json:"targetDbId"`     // 目标库id
	TargetDbName   string `json:"targetDbName"`   // 目标库名
	TargetDbType   string `json:"targetDbType"`   // 目标库类型
	TargetInstName string `json:"targetInstName"` // 目标库实例名
	TargetTagPath  string `json:"targetTagPath"`  // 目标库tagPath
}
