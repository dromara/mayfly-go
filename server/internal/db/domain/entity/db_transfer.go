package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type DbTransferTask struct {
	model.Model

	RunningState int `orm:"column(running_state)" json:"runningState"` // 运行状态 1运行中  2待运行

	CheckedKeys string `orm:"column(checked_keys)" json:"checkedKeys"` // 选中需要迁移的表
	DeleteTable int    `orm:"column(delete_table)" json:"deleteTable"` // 创建表前是否删除表
	NameCase    int    `orm:"column(name_case)"    json:"nameCase"`    // 表名、字段大小写转换  1无  2大写  3小写
	Strategy    int    `orm:"column(strategy)"     json:"strategy"`    // 迁移策略  1全量  2增量

	SrcDbId     int64  `orm:"column(src_db_id)"     json:"srcDbId"`     // 源库id
	SrcDbName   string `orm:"column(src_db_name)"   json:"srcDbName"`   // 源库名
	SrcTagPath  string `orm:"column(src_tag_path)"  json:"srcTagPath"`  // 源库tagPath
	SrcDbType   string `orm:"column(src_db_type)"   json:"srcDbType"`   // 源库类型
	SrcInstName string `orm:"column(src_inst_name)" json:"srcInstName"` // 源库实例名

	TargetDbId     int    `orm:"column(target_db_id)"     json:"targetDbId"`     // 目标库id
	TargetDbName   string `orm:"column(target_db_name)"   json:"targetDbName"`   // 目标库名
	TargetDbType   string `orm:"column(target_tag_path)"  json:"targetDbType"`   // 目标库类型
	TargetInstName string `orm:"column(target_db_type)"   json:"targetInstName"` // 目标库实例名
	TargetTagPath  string `orm:"column(target_inst_name)" json:"targetTagPath"`  // 目标库tagPath

}

func (d *DbTransferTask) TableName() string {
	return "t_db_transfer_task"
}

type DbTransferLog struct {
	model.IdModel
	TaskId      uint64     `orm:"column(task_id)" json:"taskId"` // 任务表id
	CreateTime  *time.Time `orm:"column(create_time)" json:"createTime"`
	DataSqlFull string     `orm:"column(data_sql_full)" json:"dataSqlFull"` // 执行的完整sql
	ResNum      int        `orm:"column(res_num)" json:"resNum"`            // 收到数据条数
	ErrText     string     `orm:"column(err_text)" json:"errText"`          // 错误日志
	Status      int8       `orm:"column(status)" json:"status"`             // 状态:1.成功  -1.失败
}

func (d *DbTransferLog) TableName() string {
	return "t_db_transfer_log"
}

const (
	DbTransferTaskStatusEnable  int = 1  // 启用状态
	DbTransferTaskStatusDisable int = -1 // 禁用状态

	DbTransferTaskStateSuccess int = 1  // 执行成功状态
	DbTransferTaskStateRunning int = 2  // 执行成功状态
	DbTransferTaskStateFail    int = -1 // 执行失败状态

	DbTransferTaskRunStateRunning int = 1 // 运行中状态
	DbTransferTaskRunStateStop    int = 2 // 手动停止状态
)
