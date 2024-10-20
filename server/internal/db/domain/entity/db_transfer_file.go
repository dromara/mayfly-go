package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type DbTransferFile struct {
	model.IdModel
	IsDeleted  int8       `orm:"column(is_deleted)" json:"-"`            // 是否删除 1是 0否
	CreateTime *time.Time `orm:"column(create_time)" json:"createTime"`  // 创建时间,默认当前时间戳
	Status     int8       `orm:"column(status)" json:"status"`           // 状态 1、执行中 2、执行成功 3、执行失败
	TaskId     uint64     `orm:"column(task_id)" json:"taskId"`          // 迁移任务ID
	LogId      uint64     `orm:"column(log_id)" json:"logId"`            // 日志ID
	FileDbType string     `orm:"column(file_db_type)" json:"fileDbType"` // sql文件数据库类型
	FileName   string     `orm:"column(file_name)" json:"fileName"`      // 显式文件名
	FileUuid   string     `orm:"column(file_uuid)" json:"fileUuid"`      // 文件真实id，拼接后可以下载
}

func (d *DbTransferFile) TableName() string {
	return "t_db_transfer_files"
}

const (
	DbTransferFileStatusRunning int8 = 1
	DbTransferFileStatusSuccess int8 = 2
	DbTransferFileStatusFail    int8 = -1
)
