package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// DbRestoreHistory 数据库恢复历史
type DbRestoreHistory struct {
	model.DeletedModel

	CreateTime  time.Time `orm:"column(create_time)" json:"createTime"` // 创建时间: 2023-11-08 02:00:00
	DbRestoreId uint64    `orm:"column(db_restore_id)" json:"dbRestoreId"`
}

func (d *DbRestoreHistory) TableName() string {
	return "t_db_restore_history"
}
