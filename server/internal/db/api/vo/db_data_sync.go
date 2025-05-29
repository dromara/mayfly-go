package vo

import "time"

type DataSyncTaskListVO struct {
	Id           int64      `json:"id"`
	TaskName     string     `json:"taskName"`
	TaskCron     string     `json:"cron"`
	CreateTime   *time.Time `json:"createTime"`
	Creator      string     `json:"creator"`
	UpdateTime   *time.Time `json:"updateTime"`
	ModifierId   uint64     `json:"modifierId"`
	Modifier     string     `json:"modifier"`
	RecentState  int        `json:"recentState"`
	RunningState int        `json:"runningState"`
	Status       int        `json:"status"`
}

type DataSyncLogListVO struct {
	CreateTime  *time.Time `json:"createTime"`
	DataSqlFull string     `json:"dataSqlFull"`
	ResNum      int        `json:"resNum"`
	ErrText     string     `json:"errText"`
	Status      *int       `json:"status"`
}
