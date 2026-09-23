package vo

import (
	"mayfly-go/internal/db/domain/entity"
	"time"
)

type DataSyncTaskListVO struct {
	Id           int64               `json:"id"`
	TaskName     string              `json:"taskName"`
	TaskCron     string              `json:"cron"`
	SyncMode     entity.DataSyncMode `json:"syncMode"`
	CreateTime   *time.Time          `json:"createTime"`
	Creator      string              `json:"creator"`
	UpdateTime   *time.Time          `json:"updateTime"`
	ModifierId   uint64              `json:"modifierId"`
	Modifier     string              `json:"modifier"`
	RecentState  int                 `json:"recentState"`
	RunningState int                 `json:"runningState"`
	Status       int                 `json:"status"`

	// 源/目标基本信息
	SrcDbId         int64  `json:"srcDbId"`
	SrcDbName       string `json:"srcDbName"`
	TargetDbId      int64  `json:"targetDbId"`
	TargetDbName    string `json:"targetDbName"`
	TargetTableName string `json:"targetTableName"`

	// 双向同步
	BiDirEnabled  bool   `json:"biDirEnabled"`
	ReverseTaskId uint64 `json:"reverseTaskId"`
}

// DataSyncLogListVO 数据同步执行日志列表行。
// 不含 runLog/dataSqlFull：两者为 text 大字段，列表一次返回多条会使响应体膨胀至数百 KB，
// 运行日志由按日志 id 的接口单条获取
type DataSyncLogListVO struct {
	Id         uint64     `json:"id"`
	CreateTime *time.Time `json:"createTime"`
	TaskId     uint64     `json:"taskId"`
	ResNum     int        `json:"resNum"`
	ErrText    string     `json:"errText"`
	Status     *int       `json:"status"`

	// 监控指标
	DurationMs    int64 `json:"durationMs"`
	Throughput    int   `json:"throughput"`
	BatchCount    int   `json:"batchCount"`
	InsertCount   int   `json:"insertCount"`
	UpdateCount   int   `json:"updateCount"`
	DeleteCount   int   `json:"deleteCount"`
	SkipCount     int   `json:"skipCount"`
	SchemaChanges int   `json:"schemaChanges"`
}

// DataSyncLogRunVO 单条数据同步执行日志的运行日志内容
type DataSyncLogRunVO struct {
	Id     uint64 `json:"id"`
	Status int8   `json:"status"`
	RunLog string `json:"runLog"`
}
