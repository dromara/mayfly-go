package entity

import (
	"mayfly-go/pkg/model"
)

type DbTransferTask struct {
	model.Model

	RunningState     int8   `orm:"column(running_state)" json:"runningState"` // 运行状态
	LogId            uint64 `json:"logId"`
	TaskName         string `orm:"column(task_name)" json:"taskName"`                   // 任务名称
	Status           int8   `orm:"column(status)" json:"status"`                        // 启用状态 1启用 -1禁用
	CronAble         int8   `orm:"column(cron_able)" json:"cronAble"`                   // 是否定时  1是 -1否
	Cron             string `orm:"column(cron)" json:"cron"`                            // 定时任务cron表达式
	Mode             int8   `orm:"column(mode)" json:"mode"`                            // 数据迁移方式，1、迁移到数据库  2、迁移到文件
	TargetFileDbType string `orm:"column(target_file_db_type)" json:"targetFileDbType"` // 目标文件数据库类型
	TaskKey          string `orm:"column(key)" json:"taskKey"`                          // 定时任务唯一uuid key

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

const (
	DbTransferTaskStatusEnable  int8 = 1  // 启用状态
	DbTransferTaskStatusDisable int8 = -1 // 禁用状态

	DbTransferTaskCronAbleEnable  int8 = 1  // 是否定时  1是
	DbTransferTaskCronAbleDisable int8 = -1 // 是否定时  -1否

	DbTransferTaskModeDb   int8 = 1 // 数据迁移方式，1、迁移到数据库
	DbTransferTaskModeFile int8 = 2 // 数据迁移方式，2、迁移到文件

	DbTransferTaskRunStateSuccess int8 = 2  // 执行成功
	DbTransferTaskRunStateRunning int8 = 1  // 运行中状态
	DbTransferTaskRunStateFail    int8 = -1 // 执行失败
	DbTransferTaskRunStateStop    int8 = -2 // 手动终止
)
