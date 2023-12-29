package vo

// DbRestoreHistory 数据库备份历史
type DbRestoreHistory struct {
	Id          uint64 `json:"id"`
	DbRestoreId uint64 `json:"dbRestoreId"`
}
