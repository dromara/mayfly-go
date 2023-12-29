package entity

var _ DbTask = (*DbBackup)(nil)

// DbBackup 数据库备份任务
type DbBackup struct {
	*DbTaskBase

	Name         string `json:"name"`         // 备份任务名称
	DbName       string `json:"dbName"`       // 数据库名
	DbInstanceId uint64 `json:"dbInstanceId"` // 数据库实例ID
}
