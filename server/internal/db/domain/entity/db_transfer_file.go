package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// DbTransferFile 数据库迁移文件管理
type DbTransferFile struct {
	model.IdModel
	IsDeleted  int8       `json:"-" gorm:"default:0;"`                                    // 是否删除 1是 0否
	CreateTime *time.Time `json:"createTime"`                                             // 创建时间,默认当前时间戳
	Status     int8       `json:"status" gorm:"default:1;comment:状态 1、执行中 2、执行成功 3、执行失败"` // 状态 1、执行中 2、执行成功 3、执行失败
	TaskId     uint64     `json:"taskId" gorm:"comment:迁移任务ID"`                           // 迁移任务ID
	LogId      uint64     `json:"logId" gorm:"comment:日志ID"`                              // 日志ID
	FileDbType string     `json:"fileDbType" gorm:"size:32;comment:sql文件数据库类型"`           // sql文件数据库类型
	FileKey    string     `json:"fileKey" gorm:"size:50;comment:文件"`                      // 文件
}

func (d *DbTransferFile) TableName() string {
	return "t_db_transfer_files"
}

const (
	DbTransferFileStatusRunning int8 = 1
	DbTransferFileStatusSuccess int8 = 2
	DbTransferFileStatusFail    int8 = -1
)
