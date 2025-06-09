package entity

import "mayfly-go/pkg/model"

// InstanceQuery 数据库实例查询
type InstanceQuery struct {
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

	Name   string `json:"name" form:"name"`
	Status int8   `json:"status" form:"status"`
}
type DataSyncLogQuery struct {
	model.PageParam

	TaskId uint64 `json:"task_id" form:"taskId"`
}

type DbTransferTaskQuery struct {
	model.PageParam

	Name     string `json:"name" form:"name"`
	Status   int8   `json:"status" form:"status"`
	CronAble int8   `json:"cronAble" form:"cronAble"`
}
type DbTransferFileQuery struct {
	model.PageParam

	TaskId uint64 `json:"task_id" form:"taskId"`
	Name   string `json:"name" form:"name"`
}

type DbTransferLogQuery struct {
	TaskId uint64 `json:"task_id" form:"taskId"`
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

type DbSqlExecQuery struct {
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

// DbBackupQuery 数据库备份任务查询
type DbBackupQuery struct {
	Id           uint64   `json:"id" form:"id"`
	DbName       string   `json:"dbName" form:"dbName"`
	IntervalDay  int      `json:"intervalDay" form:"intervalDay"`
	InDbNames    []string `json:"-" form:"-"`
	DbInstanceId uint64   `json:"-" form:"-"`
	Repeated     bool     `json:"repeated" form:"repeated"` // 是否重复执行
}

// DbBackupHistoryQuery 数据库备份任务查询
type DbBackupHistoryQuery struct {
	Id           uint64   `json:"id" form:"id"`
	DbBackupId   uint64   `json:"dbBackupId" form:"dbBackupId"`
	DbId         string   `json:"dbId" form:"dbId"`
	DbName       string   `json:"dbName" form:"dbName"`
	InDbNames    []string `json:"-" form:"-"`
	DbInstanceId uint64   `json:"dbInstanceId" form:"dbInstanceId"`
}

// DbRestoreQuery 数据库备份任务查询
type DbRestoreQuery struct {
	*model.PageParam

	Id           uint64   `json:"id" form:"id"`
	DbName       string   `json:"dbName" form:"dbName"`
	InDbNames    []string `json:"-" form:"-"`
	DbInstanceId uint64   `json:"-" form:"-"`
	Repeated     bool     `json:"repeated" form:"repeated"` // 是否重复执行
}

// DbRestoreHistoryQuery 数据库备份任务查询
type DbRestoreHistoryQuery struct {
	Id          uint64 `json:"id" form:"id"`
	DbRestoreId uint64 `json:"dbRestoreId" form:"dbRestoreId"`
}
