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

type DataSyncLogListVO struct {
	CreateTime  *time.Time `json:"createTime"`
	DataSqlFull string     `json:"dataSqlFull"`
	ResNum      int        `json:"resNum"`
	ErrText     string     `json:"errText"`
	Status      *int       `json:"status"`

	// Phase 4: 监控指标
	DurationMs    int64 `json:"durationMs"`
	Throughput    int   `json:"throughput"`
	BatchCount    int   `json:"batchCount"`
	InsertCount   int   `json:"insertCount"`
	UpdateCount   int   `json:"updateCount"`
	DeleteCount   int   `json:"deleteCount"`
	SkipCount     int   `json:"skipCount"`
	SchemaChanges int   `json:"schemaChanges"`

	// 运行日志
	RunLog string `json:"runLog"`
}
