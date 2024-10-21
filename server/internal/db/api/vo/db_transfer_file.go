package vo

import "time"

type DbTransferFileListVO struct {
	Id         *int64     `json:"id"`
	CreateTime *time.Time `json:"createTime"`
	Status     int8       `json:"status"`
	FileDbType string     `json:"fileDbType"`
	FileKey    string     `json:"fileKey"`
	LogId      uint64     `json:"logId"` // 日志ID
}
