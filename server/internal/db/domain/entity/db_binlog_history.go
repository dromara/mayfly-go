package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// DbBinlogHistory 数据库 binlog 历史
type DbBinlogHistory struct {
	model.DeletedModel

	CreateTime     time.Time `json:"createTime"` // 创建时间: 2023-11-08 02:00:00
	FileName       string
	FileSize       int64
	Sequence       int64
	FirstEventTime time.Time
	LastEventTime  time.Time
	DbInstanceId   uint64 `json:"dbInstanceId"`
}

func (d *DbBinlogHistory) TableName() string {
	return "t_db_binlog_history"
}

type BinlogInfo struct {
	FileName string `json:"fileName"`
	Sequence int64  `json:"sequence"`
	Position int64  `json:"position"`
}
