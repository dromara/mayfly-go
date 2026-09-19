package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// DataSyncMode 数据同步模式
type DataSyncMode int8

const (
	// DataSyncModeIncrementalAppend 增量追加：仅 INSERT 新数据（默认，兼容旧行为）
	DataSyncModeIncrementalAppend DataSyncMode = 1
	// DataSyncModeIncrementalMerge 增量合并：UPSERT（INSERT + UPDATE，各方言原生语法）
	DataSyncModeIncrementalMerge DataSyncMode = 2
	// DataSyncModeFullRefresh 全量刷新：GenTruncate + 全量 INSERT
	DataSyncModeFullRefresh DataSyncMode = 3
	// DataSyncModeIncrementalSoftDel 增量+软删除：同步删除标记字段到目标表
	DataSyncModeIncrementalSoftDel DataSyncMode = 4
	// DataSyncModeIncrementalHardDel 增量+硬删除：对比源/目标主键，GenBatchDelete 清理
	DataSyncModeIncrementalHardDel DataSyncMode = 5
	// DataSyncModeValidation 数据校验：对比源/目标行数与抽样 checksum
	DataSyncModeValidation DataSyncMode = 6
)

// ConflictStrategy 双向同步冲突仲裁策略
type ConflictStrategy int8

const (
	// ConflictStrategySourceWins 源优先（默认）
	ConflictStrategySourceWins ConflictStrategy = 1
	// ConflictStrategyTargetWins 目标优先
	ConflictStrategyTargetWins ConflictStrategy = 2
	// ConflictStrategySkip 冲突跳过
	ConflictStrategySkip ConflictStrategy = 3
)

// NullStrategy 空值处理策略
type NullStrategy int8

const (
	// NullStrategyPass 保持 NULL（默认）
	NullStrategyPass NullStrategy = 0
	// NullStrategyDefault 替换为默认值
	NullStrategyDefault NullStrategy = 1
	// NullStrategySkipRow 跳过该行
	NullStrategySkipRow NullStrategy = 2
)

// SchemaEvolveMode Schema 演化检测模式
type SchemaEvolveMode int8

const (
	// SchemaEvolveOff 不检测
	SchemaEvolveOff SchemaEvolveMode = 0
	// SchemaEvolveWarn 仅告警（日志记录）
	SchemaEvolveWarn SchemaEvolveMode = 1
	// SchemaEvolveAuto 自动适配（跳过缺失列并告警）
	SchemaEvolveAuto SchemaEvolveMode = 2
)

// DataSyncTask 数据同步
type DataSyncTask struct {
	model.Model

	// 基本信息
	TaskName     string             `json:"taskName" gorm:"not null;size:255;comment:任务名"`                       // 任务名
	TaskCron     string             `json:"taskCron" gorm:"not null;size:50;comment:任务Cron表达式"`                  // 任务Cron表达式
	Status       DataSyncTaskStatus `json:"status" gorm:"not null;default:1;comment:状态 1启用  2禁用"`                // 状态 1启用  2禁用
	TaskKey      string             `json:"taskKey" gorm:"size:100;comment:任务唯一标识"`                              // 任务唯一标识
	RecentState  int8               `json:"recentState" gorm:"not null;default:0;comment:最近执行状态 1成功 -1失败"`       // 最近执行状态 1成功 -1失败
	RunningState int8               `json:"runningState" gorm:"not null;default:2;comment:运行时状态 1运行中、2待运行、3已停止"` // 运行时状态 1运行中、2待运行、3已停止

	// 同步模式
	SyncMode DataSyncMode `json:"syncMode" gorm:"not null;default:1;comment:同步模式 1增量追加 2增量合并 3全量刷新 4增量+软删除 5增量+硬删除 6数据校验"` // 同步模式

	// 源数据库信息
	SrcDbId     int64  `json:"srcDbId" gorm:"not null;comment:源数据库ID"`                                                           // 源数据库ID
	SrcDbName   string `json:"srcDbName" gorm:"size:100;comment:源数据库名"`                                                          // 源数据库名
	SrcTagPath  string `json:"srcTagPath" gorm:"size:200;comment:源数据库tag路径"`                                                     // 源数据库tag路径
	DataSql     string `json:"dataSql" gorm:"not null;type:text;comment:数据查询sql"`                                                // 数据源查询sql
	PageSize    int    `json:"pageSize" gorm:"not null;comment:数据同步分页大小"`                                                        // 配置分页sql查询的条数
	UpdField    string `json:"updField" gorm:"not null;size:100;default:'id';comment:更新字段，默认'id'"`                               // 更新字段， 选择由哪个字段为更新字段，查询数据源的时候会带上这个字段，如：where update_time > {最近更新的最大值}
	UpdFieldVal string `json:"updFieldVal" gorm:"size:100;comment:当前更新值"`                                                        // 更新字段当前值
	UpdFieldSrc string `json:"updFieldSrc" gorm:"comment:更新值来源, 如select name as user_name from user;  则updFieldSrc的值为user_name"` // 更新值来源, 如select name as user_name from user;  则updFieldSrc的值为user_name

	// Phase 2: 多字段增量条件（辅助增量字段，与 UpdField 联合构建 AND 条件）
	UpdFieldSecondary string `json:"updFieldSecondary" gorm:"size:200;comment:辅助增量字段"` // 辅助增量字段（如 update_time），与主增量字段联合使用

	// Phase 3: 数据转换与过滤
	TransformRules  string       `json:"transformRules" gorm:"type:text;comment:字段转换规则json"`               // [{"targetColumn":"col","type":"expr","config":{"expression":"UPPER(src.col)"}}]
	FilterCondition string       `json:"filterCondition" gorm:"size:500;comment:数据过滤条件"`                   // Go 层过滤条件表达式（如 "status == 1"）
	NullStrategy    NullStrategy `json:"nullStrategy" gorm:"not null;default:0;comment:空值策略 0保持 1默认值 2跳过"` // 空值处理策略
	NullDefault     string       `json:"nullDefault" gorm:"size:200;comment:空值默认值"`                        // NullStrategy=1 时的默认值

	// 删除同步配置（SyncMode为4/5时使用）
	SoftDeleteField string `json:"softDeleteField" gorm:"size:100;comment:软删除标记源字段"` // 软删除标记源字段名
	SoftDeleteValue string `json:"softDeleteValue" gorm:"size:100;comment:软删除标记值"`   // 软删除标记值（如 "1" 表示已删除）

	// 目标数据库信息
	TargetDbId        int64  `json:"targetDbId" gorm:"not null;comment:目标数据库ID"`                                  // 目标数据库ID
	TargetDbName      string `json:"targetDbName" gorm:"size:150;comment:目标数据库名"`                                 // 目标数据库名
	TargetTagPath     string `json:"targetTagPath" gorm:"size:255;comment:目标数据库tag路径"`                            // 目标数据库tag路径
	TargetTableName   string `json:"targetTableName" gorm:"size:150;comment:目标数据库表名"`                             // 目标数据库表名
	FieldMap          string `json:"fieldMap" gorm:"type:text;comment:字段映射json"`                                  // 字段映射json
	DuplicateStrategy int    `json:"duplicateStrategy" gorm:"not null;default:-1;comment:唯一键冲突策略 -1：无，1：忽略，2：覆盖"` // 冲突策略 -1：无，1：忽略，2：覆盖

	// Phase 5: Schema 演化感知
	SchemaEvolveMode SchemaEvolveMode `json:"schemaEvolveMode" gorm:"not null;default:0;comment:Schema演化检测 0关 1告警 2自动适配"` // Schema 变更检测模式

	// Phase 6.3: 双向同步
	BiDirEnabled        bool             `json:"biDirEnabled" gorm:"not null;default:false;comment:是否启用双向同步"`            // 是否启用双向同步
	ReverseTaskId       uint64           `json:"reverseTaskId" gorm:"comment:反向同步任务ID"`                                  // 反向同步任务 ID（0 表示未创建）
	ConflictStrategy    ConflictStrategy `json:"conflictStrategy" gorm:"not null;default:1;comment:冲突策略 1源优先 2目标优先 3跳过"` // 冲突仲裁策略
	BiDirTimestampField string           `json:"biDirTimestampField" gorm:"size:100;comment:双向同步时间戳比较字段"`                // 用于冲突检测的时间戳字段
}

func (d *DataSyncTask) TableName() string {
	return "t_db_data_sync_task"
}

// DataSyncLog 同步执行日志（含 Phase 4 监控指标字段）
type DataSyncLog struct {
	model.IdModel

	CreateTime  *time.Time `json:"createTime" gorm:"not null;"`                            // 创建时间
	TaskId      uint64     `json:"taskId" gorm:"not null;comment:同步任务表id"`                 // 任务表id
	DataSqlFull string     `json:"dataSqlFull" gorm:"not null;type:text;comment:执行的完整sql"` // 执行的完整sql
	ResNum      int        `json:"resNum" gorm:"comment:收到数据条数"`                           // 收到数据条数
	ErrText     string     `json:"errText" gorm:"type:text;comment:日志"`                    // 日志
	Status      int8       `json:"status" gorm:"not null;default:1;comment:状态:1.成功  0.失败"` // 状态:1.成功  0.失败

	// Phase 4: 监控指标
	DurationMs    int64 `json:"durationMs" gorm:"comment:执行耗时(毫秒)"`     // 执行耗时（毫秒）
	Throughput    int   `json:"throughput" gorm:"comment:吞吐量(行/秒)"`     // 吞吐量（rows/sec）
	BatchCount    int   `json:"batchCount" gorm:"comment:批次数"`          // 执行批次数
	InsertCount   int   `json:"insertCount" gorm:"comment:新增行数"`        // INSERT 行数
	UpdateCount   int   `json:"updateCount" gorm:"comment:更新行数"`        // UPDATE 行数（UPSERT 中实际更新的部分）
	DeleteCount   int   `json:"deleteCount" gorm:"comment:删除行数"`        // DELETE 行数（硬删除模式）
	SkipCount     int   `json:"skipCount" gorm:"comment:跳过行数"`          // 跳过行数（过滤/冲突/空值策略）
	SchemaChanges int   `json:"schemaChanges" gorm:"comment:Schema变更数"` // 本次检测到的 Schema 变更数

	// 运行日志：追加式执行过程记录（类似数据迁移的 AppendLog 机制）
	RunLog string `json:"runLog" gorm:"type:text;comment:运行日志"` // 运行日志（追加式）
}

func (d *DataSyncLog) TableName() string {
	return "t_db_data_sync_log"
}

// DataSyncTaskQuery 数据同步任务状态
type DataSyncTaskStatus int8

const (
	DataSyncTaskStatusEnable  DataSyncTaskStatus = 1  // 启用状态
	DataSyncTaskStatusDisable DataSyncTaskStatus = -1 // 禁用状态

	DataSyncTaskStateSuccess int8 = 1  // 执行成功状态
	DataSyncTaskStateRunning int8 = 2  // 执行中状态
	DataSyncTaskStateFail    int8 = -1 // 执行失败状态

	DataSyncTaskRunStateRunning int8 = 1 // 运行中状态
	DataSyncTaskRunStateReady   int8 = 2 // 待运行状态
	DataSyncTaskRunStateStop    int8 = 3 // 手动停止状态
)

// SyncDirection 同步方向（用于双向同步标识）
type SyncDirection int8

const (
	SyncDirectionForward SyncDirection = 1 // 正向同步
	SyncDirectionReverse SyncDirection = 2 // 反向同步
)
