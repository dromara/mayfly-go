package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type DbTransferTask struct {
	model.Model
	model.ExtraData

	TaskName         string              `json:"taskName" gorm:"size:255;not null;"`      // 任务名称
	TaskKey          string              `json:"taskKey" gorm:"size:100;not null;"`       // 定时任务唯一uuid key
	CronEnabled      TransferCronEnabled `json:"cronEnabled" gorm:"default:-1;not null;"` // 是否定时  1是 -1否
	Cron             string              `json:"cron" gorm:"size:32;"`                    // 定时任务cron表达式
	Mode             TransferMode        `json:"mode"`                                    // 数据迁移方式，1、迁移到数据库  2、迁移到文件
	TargetFileDbType string              `json:"targetFileDbType" gorm:"size:32;"`        // 目标文件数据库类型
	FileSaveDays     int                 `json:"fileSaveDays"`                            // 文件保存天数
	Status           TransferTaskStatus  `json:"status"`                                  // 启用状态 1启用 -1禁用
	RunningState     TransferRunState    `json:"runningState"`                            // 运行状态
	LogId            uint64              `json:"logId"`

	CheckedKeys string `json:"checkedKeys" gorm:"type:text;"` // 选中需要迁移的表
	DeleteTable int8   `json:"deleteTable"`                   // 创建表前是否删除表
	NameCase    int8   `json:"nameCase"`                      // 表名、字段大小写转换  1无  2大写  3小写
	Strategy    int8   `json:"strategy"`                      // 迁移策略  1全量  2增量
	Concurrency int    `json:"concurrency" gorm:"default:4;"` // 迁移并行度（数据导入工作池大小），0=默认4，范围1~16

	SrcDbId     int64  `json:"srcDbId" gorm:"not null;"`            // 源库id
	SrcDbName   string `json:"srcDbName" gorm:"size:255;not null;"` // 源库名
	SrcTagPath  string `json:"srcTagPath" gorm:"size:255;"`         // 源库tagPath
	SrcDbType   string `json:"srcDbType" gorm:"size:32;not null;"`  // 源库类型
	SrcInstName string `json:"srcInstName" gorm:"size:255;"`        // 源库实例名

	TargetDbId     int    `json:"targetDbId" gorm:"not null;"`            // 目标库id
	TargetDbName   string `json:"targetDbName" gorm:"size:255;not null;"` // 目标库名
	TargetDbType   string `json:"targetDbType" gorm:"size:32;not null;"`  // 目标库类型
	TargetInstName string `json:"targetInstName" gorm:"size:255;"`        // 目标库实例名
	TargetTagPath  string `json:"targetTagPath" gorm:"size:255;"`         // 目标库tagPath
}

func (d *DbTransferTask) TableName() string {
	return "t_db_transfer_task"
}

// DbTransferCheckpoint 迁移任务断点续传检查点（每个任务至多一条记录）。
// 独立成表而非存任务Extra：Extra为varchar(2000)，全库迁移时表名清单易超出容量。
type DbTransferCheckpoint struct {
	model.IdModel

	TaskId        uint64     `json:"taskId" gorm:"uniqueIndex;not null;comment:迁移任务id"`   // 迁移任务id
	PlannedTables string     `json:"plannedTables" gorm:"type:text;comment:计划迁移表名JSON数组"` // 计划迁移表名JSON数组（启动时快照，保证续传确定性）
	DoneTables    string     `json:"doneTables" gorm:"type:text;comment:已完成表名JSON数组"`     // 已完成表名JSON数组
	CreateTime    *time.Time `json:"createTime" gorm:"comment:创建时间"`                      // 创建时间
	UpdateTime    *time.Time `json:"updateTime" gorm:"comment:更新时间"`                      // 更新时间
}

func (d *DbTransferCheckpoint) TableName() string {
	return "t_db_transfer_checkpoint"
}

const (
	DbTransferTaskStatusEnable  TransferTaskStatus = 1  // 启用状态
	DbTransferTaskStatusDisable TransferTaskStatus = -1 // 禁用状态

	DbTransferTaskCronEnabled  TransferCronEnabled = 1  // 是否定时  1是
	DbTransferTaskCronDisabled TransferCronEnabled = -1 // 是否定时  -1否

	DbTransferTaskModeDb   TransferMode = 1 // 数据迁移方式，1、迁移到数据库
	DbTransferTaskModeFile TransferMode = 2 // 数据迁移方式，2、迁移到文件

	DbTransferTaskRunStateSuccess TransferRunState = 2  // 执行成功
	DbTransferTaskRunStateRunning TransferRunState = 1  // 运行中状态
	DbTransferTaskRunStateFail    TransferRunState = -1 // 执行失败
	DbTransferTaskRunStateStop    TransferRunState = -2 // 手动终止

	// 建表前是否删除表（DeleteTable）：1是（生成 DROP，默认） 2否（保留已有表）
	DbTransferTaskDeleteTableYes int8 = 1
	DbTransferTaskDeleteTableNo  int8 = 2

	// 表名/字段名大小写转换（NameCase）：1无 2大写 3小写
	DbTransferTaskNameCaseNone  int8 = 1
	DbTransferTaskNameCaseUpper int8 = 2
	DbTransferTaskNameCaseLower int8 = 3
)

// TransferTaskStatus 迁移任务启用状态
type TransferTaskStatus int8

// TransferCronEnabled 迁移任务定时开关
type TransferCronEnabled int8

// TransferMode 迁移模式
type TransferMode int8

// TransferRunState 迁移任务运行状态
type TransferRunState int8

// DbTransferLog 迁移任务执行日志（对齐 DataSyncLog 架构）。
// 每次执行生成独立记录，支持历史查询与指标统计。
type DbTransferLog struct {
	model.IdModel

	CreateTime *time.Time           `json:"createTime" gorm:"not null;"`                                  // 创建时间
	TaskId     uint64               `json:"taskId" gorm:"not null;index;comment:迁移任务id"`                  // 迁移任务id
	Mode       TransferMode         `json:"mode" gorm:"not null;comment:迁移模式 1数据库 2文件"`                   // 迁移模式
	Purpose    DbTransferLogPurpose `json:"purpose" gorm:"not null;default:1;comment:执行用途 1迁移 2导出文件 3校验"` // 执行用途：区分同一任务下不同性质的执行记录
	TargetFile string               `json:"targetFile" gorm:"size:255;comment:目标文件名"`                     // 目标文件名（文件迁移模式）
	ErrText    string               `json:"errText" gorm:"type:text;comment:错误信息"`                        // 错误信息
	Status     int8                 `json:"status" gorm:"not null;default:2;comment:状态:2.执行中 1.成功 -1.失败"` // 状态:2.执行中 1.成功 -1.失败

	// 监控指标
	DurationMs int64 `json:"durationMs" gorm:"comment:执行耗时(毫秒)"` // 执行耗时（毫秒）
	TotalRows  int64 `json:"totalRows" gorm:"comment:迁移总行数"`     // 迁移总行数
	TableCount int   `json:"tableCount" gorm:"comment:迁移表数"`     // 迁移表数

	// 运行日志：追加式执行过程记录
	RunLog string `json:"runLog" gorm:"type:text;comment:运行日志"` // 运行日志（追加式）
}

func (d *DbTransferLog) TableName() string {
	return "t_db_transfer_log"
}

// 迁移任务执行日志状态
const (
	DbTransferLogStatusRunning int8 = 2 // 执行中
	DbTransferLogStatusSuccess int8 = 1 // 执行成功
	// DbTransferLogStatusFail 执行失败，取值必须为非零值：日志收尾以结构体更新落库，
	// GORM 会跳过零值字段，失败态取 0 将永远写不进库，日志会一直停留在「执行中」
	DbTransferLogStatusFail int8 = -1
)

// DbTransferLogPurpose 迁移执行日志的用途。
// Mode 是任务级属性（迁移到库/文件），Purpose 是执行级属性：同一库模式任务既可发起迁移、
// 也可发起只读校验，需据此区分日志记录性质，避免校验记录被误当成迁移（行数/表数恒为 0）。
type DbTransferLogPurpose int8

const (
	DbTransferLogPurposeTransfer DbTransferLogPurpose = 1 // 数据迁移（库→库）
	DbTransferLogPurposeExport   DbTransferLogPurpose = 2 // 导出文件（库→文件）
	DbTransferLogPurposeVerify   DbTransferLogPurpose = 3 // 数据校验（只读比对，不迁移数据）
)
