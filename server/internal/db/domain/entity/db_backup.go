package entity

var _ DbTask = (*DbBackup)(nil)

// DbBackup 数据库备份任务
type DbBackup struct {
	*DbTaskBase

	Name         string `json:"name"`         // 备份任务名称
	DbName       string `json:"dbName"`       // 数据库名
	DbInstanceId uint64 `json:"dbInstanceId"` // 数据库实例ID
}

func (*DbBackup) MessageWithStatus(status TaskStatus) string {
	var result string
	switch status {
	case TaskDelay:
		result = "等待备份数据库"
	case TaskReady:
		result = "准备备份数据库"
	case TaskReserved:
		result = "数据库备份中"
	case TaskSuccess:
		result = "数据库备份成功"
	case TaskFailed:
		result = "数据库备份失败"
	}
	return result
}
