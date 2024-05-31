package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// DbBackupHistory 数据库备份历史
type DbBackupHistory struct {
	model.DeletedModel

	Uuid           string    `json:"uuid"`
	Name           string    `json:"name"`       // 备份历史名称
	CreateTime     time.Time `json:"createTime"` // 创建时间: 2023-11-08 02:00:00
	DbBackupId     uint64    `json:"dbBackupId"`
	DbInstanceId   uint64    `json:"dbInstanceId"`
	DbName         string    `json:"dbName"`
	BinlogFileName string    `json:"binlogFileName"`
	BinlogSequence int64     `json:"binlogSequence"`
	BinlogPosition int64     `json:"binlogPosition"`
}

func (d *DbBackupHistory) TableName() string {
	return "t_db_backup_history"
}
