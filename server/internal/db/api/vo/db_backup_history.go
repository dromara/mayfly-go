package vo

import "time"

// DbBackupHistory 数据库备份历史
type DbBackupHistory struct {
	Id         uint64    `json:"id"`
	DbBackupId uint64    `json:"dbBackupId"`
	CreateTime time.Time `json:"createTime"`
	DbName     string    `json:"dbName"` // 数据库名称
	Name       string    `json:"name"`   // 备份历史名称
}
