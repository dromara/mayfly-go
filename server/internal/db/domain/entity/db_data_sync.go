package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type DataSyncTask struct {
	model.Model

	// 基本信息
	TaskName     string `orm:"column(task_name)" json:"taskName"`         // 任务名
	TaskCron     string `orm:"column(task_cron)" json:"taskCron"`         // 任务Cron表达式
	Status       int8   `orm:"column(status)" json:"status"`              // 状态 1启用  2禁用
	TaskKey      string `orm:"column(key)" json:"taskKey"`                // 任务唯一标识
	RecentState  int8   `orm:"column(recent_state)" json:"recentState"`   // 最近执行状态 1成功 -1失败
	RunningState int8   `orm:"column(running_state)" json:"runningState"` // 运行时状态 1运行中、2待运行、3已停止

	// 源数据库信息
	SrcDbId     int64  `orm:"column(src_db_id)" json:"srcDbId"`
	SrcDbName   string `orm:"column(src_db_name)" json:"srcDbName"`
	SrcTagPath  string `orm:"column(src_tag_path)" json:"srcTagPath"`
	DataSql     string `orm:"column(data_sql)" json:"dataSql"`          // 数据源查询sql
	PageSize    int    `orm:"column(page_size)" json:"pageSize"`        // 配置分页sql查询的条数
	UpdField    string `orm:"column(upd_field)" json:"updField"`        //更新字段， 选择由哪个字段为更新字段，查询数据源的时候会带上这个字段，如：where update_time > {最近更新的最大值}
	UpdFieldVal string `orm:"column(upd_field_val)" json:"updFieldVal"` // 更新字段当前值

	// 目标数据库信息
	TargetDbId      int64  `orm:"column(target_db_id)" json:"targetDbId"`
	TargetDbName    string `orm:"column(target_db_name)" json:"targetDbName"`
	TargetTagPath   string `orm:"column(target_tag_path)" json:"targetTagPath"`
	TargetTableName string `orm:"column(target_table_name)" json:"targetTableName"`
	FieldMap        string `orm:"column(field_map)" json:"fieldMap"` // 字段映射json
}

func (d *DataSyncTask) TableName() string {
	return "t_db_data_sync_task"
}

type DataSyncLog struct {
	model.IdModel
	TaskId      uint64     `orm:"column(task_id)" json:"taskId"` // 任务表id
	CreateTime  *time.Time `orm:"column(create_time)" json:"createTime"`
	DataSqlFull string     `orm:"column(data_sql_full)" json:"dataSqlFull"` // 执行的完整sql
	ResNum      int        `orm:"column(res_num)" json:"resNum"`            // 收到数据条数
	ErrText     string     `orm:"column(err_text)" json:"errText"`          // 错误日志
	Status      int8       `orm:"column(status)" json:"status"`             // 状态:1.成功  -1.失败
}

func (d *DataSyncLog) TableName() string {
	return "t_db_data_sync_log"
}

const (
	DataSyncTaskStatusEnable  int8 = 1  // 启用状态
	DataSyncTaskStatusDisable int8 = -1 // 禁用状态

	DataSyncTaskStateSuccess int8 = 1  // 执行成功状态
	DataSyncTaskStateRunning int8 = 2  // 执行成功状态
	DataSyncTaskStateFail    int8 = -1 // 执行失败状态

	DataSyncTaskRunStateRunning int8 = 1 // 运行中状态
	DataSyncTaskRunStateReady   int8 = 2 // 待运行状态
	DataSyncTaskRunStateStop    int8 = 3 // 手动停止状态
)
