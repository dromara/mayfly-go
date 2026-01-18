package entity

import (
	"mayfly-go/pkg/model"
)

type DbTransferTask struct {
	model.Model
	model.ExtraData

	TaskName         string `json:"taskName" gorm:"size:255;not null;"`   // 任务名称
	TaskKey          string `json:"taskKey" gorm:"size:100;not null;"`    // 定时任务唯一uuid key
	CronAble         int8   `json:"cronAble" gorm:"default:-1;not null;"` // 是否定时  1是 -1否
	Cron             string `json:"cron" gorm:"size:32;"`                 // 定时任务cron表达式
	Mode             int8   `json:"mode"`                                 // 数据迁移方式，1、迁移到数据库  2、迁移到文件
	TargetFileDbType string `json:"targetFileDbType" gorm:"size:32;"`     // 目标文件数据库类型
	FileSaveDays     int    `json:"fileSaveDays"`                         // 文件保存天数
	Status           int8   `json:"status"`                               // 启用状态 1启用 -1禁用
	RunningState     int8   `json:"runningState"`                         // 运行状态
	LogId            uint64 `json:"logId"`

	CheckedKeys string `json:"checkedKeys" gorm:"type:text;"` // 选中需要迁移的表
	DeleteTable int8   `json:"deleteTable"`                   // 创建表前是否删除表
	NameCase    int8   `json:"nameCase"`                      // 表名、字段大小写转换  1无  2大写  3小写
	Strategy    int8   `json:"strategy"`                      // 迁移策略  1全量  2增量

	SrcDbId     int64  `json:"srcDbId" gorm:"not null;"`            // 源库id
	SrcDbName   string `json:"srcDbName" gorm:"size:255;not null;"` // 源库名
	SrcTagPath  string `json:"srcTagPath" gorm:"size:255;"`         // 源库tagPath
	SrcDbType   string `json:"srcDbType" gorm:"size:32;not null;"`  // 源库类型
	SrcInstName string `json:"srcInstName" gorm:"size:255;"`        // 源库实例名

	TargetDbId     int    `json:"targetDbId" gorm:"not null;"`            // 目标库id
	TargetDbName   string `json:"targetDbName" gorm:"size:255;not null;"` // 目标库名
	TargetDbType   string `json:"targetDbType" gorm:"size:32;not null;"`  // 目标库类型
	TargetInstName string `json:"targetInstName" gorm:"size:255;"`        // 目标库实例名
	TargetTagPath  string `json:"targetTagPath" gorm:"size:255;"`         // 目标库tagPath
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
