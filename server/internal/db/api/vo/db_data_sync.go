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
	SrcTagPath      string `json:"srcTagPath"`
	SrcDbType       string `json:"srcDbType"`
	TargetDbId      int64  `json:"targetDbId"`
	TargetDbName    string `json:"targetDbName"`
	TargetTagPath   string `json:"targetTagPath"`
	TargetTableName string `json:"targetTableName"`
	TargetDbType    string `json:"targetDbType"`

	// 增量水位：列表据此回答“同步到哪了”（仅增量追加/合并模式参与拼接增量条件）
	UpdField    string `json:"updField"`
	UpdFieldVal string `json:"updFieldVal"`

	// 写入行为配置
	PageSize          int `json:"pageSize"`
	DuplicateStrategy int `json:"duplicateStrategy"`

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
