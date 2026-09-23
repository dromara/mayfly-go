package entity

import "mayfly-go/pkg/model"

// DbInstanceQuery 数据库实例查询
type DbInstanceQuery struct {
	model.PageParam

	Id      uint64 `json:"id" form:"id"`
	Name    string `json:"name" form:"name"`
	Code    string `json:"code" form:"code"`
	Host    string `json:"host" form:"host"`
	TagPath string `json:"tagPath" form:"tagPath"`
	Keyword string `json:"keyword" form:"keyword"`
	Codes   []string
}

type DataSyncTaskQuery struct {
	model.PageParam

	Name   string             `json:"name" form:"name"`
	Status DataSyncTaskStatus `json:"status" form:"status"`
}
type DataSyncLogQuery struct {
	model.PageParam

	TaskId uint64 `json:"taskId" form:"taskId"`
}

type DbTransferTaskQuery struct {
	model.PageParam

	Name        string              `json:"name" form:"name"`
	Status      TransferTaskStatus  `json:"status" form:"status"`
	CronEnabled TransferCronEnabled `json:"cronEnabled" form:"cronEnabled"`
}
type DbTransferFileQuery struct {
	model.PageParam

	TaskId uint64 `json:"taskId" form:"taskId"`
	Name   string `json:"name" form:"name"`
}

type DbTransferLogQuery struct {
	model.PageParam

	TaskId uint64 `json:"taskId" form:"taskId"`
}

// 数据库查询实体，不与数据库表字段一一对应
type DbQuery struct {
	model.PageParam

	Id         uint64 `form:"id"`
	TagPath    string `form:"tagPath"`
	Code       string `json:"code" form:"code"`
	Codes      []string
	InstanceId uint64 `form:"instanceId"`
}

type DbSQLExecQuery struct {
	model.PageParam

	Id         uint64 `json:"id" form:"id"`
	DbId       uint64 `json:"dbId" form:"dbId"`
	Db         string `json:"db" form:"db"`
	Table      string `json:"table" form:"table"`
	Type       int8   `json:"type" form:"type"` // 类型
	FlowBizKey string `json:"flowBizKey" form:"flowBizKey"`
	Keyword    string `json:"keyword" form:"keyword"`
	StartTime  string `json:"startTime" form:"startTime"`
	EndTime    string `json:"endTime" form:"endTime"`

	Status    []int8
	CreatorId uint64
}
