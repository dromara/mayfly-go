package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// DataSyncTask 数据同步
type DataSyncTask struct {
	model.Model

	// 基本信息
	TaskName     string `json:"taskName" gorm:"not null;size:255;comment:任务名"`                       // 任务名
	TaskCron     string `json:"taskCron" gorm:"not null;size:50;comment:任务Cron表达式"`                  // 任务Cron表达式
	Status       int8   `json:"status" gorm:"not null;default:1;comment:状态 1启用  2禁用"`                // 状态 1启用  2禁用
	TaskKey      string `json:"taskKey" gorm:"size:100;comment:任务唯一标识"`                              // 任务唯一标识
	RecentState  int8   `json:"recentState" gorm:"not null;default:0;comment:最近执行状态 1成功 -1失败"`       // 最近执行状态 1成功 -1失败
	RunningState int8   `json:"runningState" gorm:"not null;default:2;comment:运行时状态 1运行中、2待运行、3已停止"` // 运行时状态 1运行中、2待运行、3已停止

	// 源数据库信息
	SrcDbId     int64  `json:"srcDbId" gorm:"not null;comment:源数据库ID"`                                                           // 源数据库ID
	SrcDbName   string `json:"srcDbName" gorm:"size:100;comment:源数据库名"`                                                          // 源数据库名
	SrcTagPath  string `json:"srcTagPath" gorm:"size:200;comment:源数据库tag路径"`                                                     // 源数据库tag路径
	DataSql     string `json:"dataSql" gorm:"not null;type:text;comment:数据查询sql"`                                                // 数据源查询sql
	PageSize    int    `json:"pageSize" gorm:"not null;comment:数据同步分页大小"`                                                        // 配置分页sql查询的条数
	UpdField    string `json:"updField" gorm:"not null;size:100;default:'id';comment:更新字段，默认'id'"`                               // 更新字段， 选择由哪个字段为更新字段，查询数据源的时候会带上这个字段，如：where update_time > {最近更新的最大值}
	UpdFieldVal string `json:"updFieldVal" gorm:"size:100;comment:当前更新值"`                                                        // 更新字段当前值
	UpdFieldSrc string `json:"updFieldSrc" gorm:"comment:更新值来源, 如select name as user_name from user;  则updFieldSrc的值为user_name"` // 更新值来源, 如select name as user_name from user;  则updFieldSrc的值为user_name

	// 目标数据库信息
	TargetDbId        int64  `json:"targetDbId" gorm:"not null;comment:目标数据库ID"`                                  // 目标数据库ID
	TargetDbName      string `json:"targetDbName" gorm:"size:150;comment:目标数据库名"`                                 // 目标数据库名
	TargetTagPath     string `json:"targetTagPath" gorm:"size:255;comment:目标数据库tag路径"`                            // 目标数据库tag路径
	TargetTableName   string `json:"targetTableName" gorm:"size:150;comment:目标数据库表名"`                             // 目标数据库表名
	FieldMap          string `json:"fieldMap" gorm:"type:text;comment:字段映射json"`                                  // 字段映射json
	DuplicateStrategy int    `json:"duplicateStrategy" gorm:"not null;default:-1;comment:唯一键冲突策略 -1：无，1：忽略，2：覆盖"` // 冲突策略 -1：无，1：忽略，2：覆盖
}

func (d *DataSyncTask) TableName() string {
	return "t_db_data_sync_task"
}

type DataSyncLog struct {
	model.IdModel

	CreateTime  *time.Time `json:"createTime" gorm:"not null;"`                            // 创建时间
	TaskId      uint64     `json:"taskId" gorm:"not null;comment:同步任务表id"`                 // 任务表id
	DataSqlFull string     `json:"dataSqlFull" gorm:"not null;type:text;comment:执行的完整sql"` // 执行的完整sql
	ResNum      int        `json:"resNum" gorm:"comment:收到数据条数"`                           // 收到数据条数
	ErrText     string     `json:"errText" gorm:"type:text;comment:日志"`                    // 日志
	Status      int8       `json:"status" gorm:"not null;default:1;comment:状态:1.成功  0.失败"` // 状态:1.成功  0.失败
}

func (d *DataSyncLog) TableName() string {
	return "t_db_data_sync_log"
}

const (
	DataSyncTaskStatusEnable  int8 = 1  // 启用状态
	DataSyncTaskStatusDisable int8 = -1 // 禁用状态

	DataSyncTaskStateSuccess int8 = 1  // 执行成功状态
	DataSyncTaskStateRunning int8 = 2  // 执行中状态
	DataSyncTaskStateFail    int8 = -1 // 执行失败状态

	DataSyncTaskRunStateRunning int8 = 1 // 运行中状态
	DataSyncTaskRunStateReady   int8 = 2 // 待运行状态
	DataSyncTaskRunStateStop    int8 = 3 // 手动停止状态
)
