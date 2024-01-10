package entity

// InstanceQuery 数据库实例查询
type InstanceQuery struct {
	Id   uint64 `json:"id" form:"id"`
	Name string `json:"name" form:"name"`
	Host string `json:"host" form:"host"`
}

type DataSyncTaskQuery struct {
	Name   string `json:"name" form:"name"`
	Status int8   `json:"status" form:"status"`
}
type DataSyncLogQuery struct {
	TaskId uint64 `json:"task_id" form:"taskId"`
}

// 数据库查询实体，不与数据库表字段一一对应
type DbQuery struct {
	Id uint64 `form:"id"`

	Name     string `orm:"column(name)" json:"name"`
	Database string `orm:"column(database)" json:"database"`
	Remark   string `json:"remark"`

	Codes   []string
	TagIds  []uint64 `orm:"column(tag_id)"`
	TagPath string   `form:"tagPath"`

	InstanceId uint64 `form:"instanceId"`
}

type DbSqlExecQuery struct {
	Id    uint64 `json:"id" form:"id"`
	DbId  uint64 `json:"dbId" form:"dbId"`
	Db    string `json:"db" form:"db"`
	Table string `json:"table" form:"table"`
	Type  int8   `json:"type" form:"type"` // 类型

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
