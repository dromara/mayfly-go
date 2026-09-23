package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// DataSyncMode 数据同步模式。
//
// 每种模式的语义不同，开启前需确认它的前提与局限（实现见 application/sync）：
//
//   - 完整性前提：能读到哪些源行、需要哪些目标列属性；
//   - 提交点：批次事务、目标写入、水位推进三者何时发生，中途失败后什么能回滞；
//   - 失败行为：目标数据与水位处于什么状态、重启后如何继续。
//
// 新模式不得默默往枚举里加，必须同步补全以上三段、并使前端 tooltip/单测同时覆盖。
type DataSyncMode int8

const (
	// DataSyncModeIncrementalAppend 增量追加：仅 INSERT 新数据（默认，兼容旧行为）。
	// 完整性前提：目标表无主键/唯一约束时行可重复；有约束时冲突行静默丢失（需搭配 DuplicateStrategy=Ignore 避免报错）。
	// 提交点：批次事务内 INSERT 成功即持久；失败批次可回滞。水位在批次提交后单独推进，与事务非同原子。
	// 失败行为：中途崩溃时本批次回滞，水位可能已推至上一批末尾；重启后从水位继续，不重补本批。
	DataSyncModeIncrementalAppend DataSyncMode = 1
	// DataSyncModeIncrementalMerge 增量合并：UPSERT（INSERT + UPDATE，各方言原生语法）。
	// 完整性前提：目标表需有主键或单一非空唯一索引作冲突键，否则退化为直插（同模式 1）。增量字段需能被业务推高，自增 id 不行。
	// 提交点：与模式 1 一致；水位推进与写入同分批次。
	// 失败行为：与模式 1 一致；重启后可能重跑同批，幂等由 UPSERT 保证。
	DataSyncModeIncrementalMerge DataSyncMode = 2
	// DataSyncModeFullRefresh 全量刷新：GenTruncate + 全量 INSERT。
	// 完整性前提：不依赖增量字段，每次读全量源行；但目标表与写入中途失败时旧数据已被清空、不能恢复。
	// 提交点：TRUNCATE 与批次写入不在同一事务——truncate 成功即不可逆；批次逐个提交。
	// 失败行为：中途失败时目标处于部分写入状态，需人工或下次任务重启从头重试（不是原子切换）。
	DataSyncModeFullRefresh DataSyncMode = 3
	// DataSyncModeIncrementalSoftDel 全量对账（源软删除）：源存在删除标记列，写入时排除已标记行，并对目标执行保留集 NOT IN 删除。
	// 完整性前提：主键集必须来自全量扫描（代码已强制 needsFullScan，不叠水位）；目标主键列非空；保留集不超上限。
	// 提交点：批次写入 + 删除对账均在目标执行；删除语句单条完成，不按批拆（拆分会退化为交集保留 → 误删）。
	// 失败行为：源结果为空会清空目标（代码仅对 clickhouse 拒绝）；删除失败时写入已完成，任务整体失败。
	DataSyncModeIncrementalSoftDel DataSyncMode = 4
	// DataSyncModeIncrementalHardDel 全量对账（源缺失删除）：与模式 4 同构，仅少了“源删除标记”这一层。
	// 完整性前提：同模式 4；尤其要求“本轮读到的所有源主键均属于保留集”——即扫描必须全量、不能被水位/分页/参数上限切断。
	// 提交点与失败行为：同模式 4。
	DataSyncModeIncrementalHardDel DataSyncMode = 5
	// DataSyncModeValidation 数据校验：对比源/目标行数 + 排序后前 100 行映射字段原值。
	// 完整性前提：不写目标。配了 TransformRules 时不执行内容比对（仅保留行数），因为源原值与目标写入值天然不等。
	// 提交点：无。失败行为：任一步骤报错即整体失败，不产生目标侧副作用。
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
	Status       DataSyncTaskStatus `json:"status" gorm:"not null;default:1;comment:状态 1启用 -1禁用"`                // 状态 1启用 -1禁用
	TaskKey      string             `json:"taskKey" gorm:"size:100;comment:任务唯一标识"`                              // 任务唯一标识
	RecentState  int8               `json:"recentState" gorm:"not null;default:0;comment:最近执行状态 1成功 -1失败"`       // 最近执行状态 1成功 -1失败
	RunningState int8               `json:"runningState" gorm:"not null;default:2;comment:运行时状态 1运行中、2待运行、3已停止"` // 运行时状态 1运行中、2待运行、3已停止

	// 同步模式
	SyncMode DataSyncMode `json:"syncMode" gorm:"not null;default:1;comment:同步模式 1增量追加 2增量合并 3全量刷新 4增量+软删除 5增量+硬删除 6数据校验"` // 同步模式

	// 源数据库信息
	SrcDbId     int64  `json:"srcDbId" gorm:"not null;comment:源数据库ID"`                                                           // 源数据库ID
	SrcDbName   string `json:"srcDbName" gorm:"size:100;comment:源数据库名"`                                                          // 源数据库名
	SrcTagPath  string `json:"srcTagPath" gorm:"size:200;comment:源数据库tag路径"`                                                     // 源数据库tag路径
	DataSQL     string `json:"dataSql" gorm:"not null;type:text;comment:数据查询sql"`                                                // 数据源查询sql
	PageSize    int    `json:"pageSize" gorm:"not null;comment:数据同步分页大小"`                                                        // 配置分页sql查询的条数
	UpdField    string `json:"updField" gorm:"not null;size:100;default:'id';comment:更新字段，默认'id'"`                               // 更新字段， 选择由哪个字段为更新字段，查询数据源的时候会带上这个字段，如：where update_time > {最近更新的最大值}
	UpdFieldVal string `json:"updFieldVal" gorm:"size:100;comment:当前更新值"`                                                        // 更新字段当前值
	UpdFieldSrc string `json:"updFieldSrc" gorm:"comment:更新值来源, 如select name as user_name from user;  则updFieldSrc的值为user_name"` // 更新值来源, 如select name as user_name from user;  则updFieldSrc的值为user_name

	// 多字段增量条件（辅助增量字段，与 UpdField 联合构建 AND 条件）
	UpdFieldSecondary string `json:"updFieldSecondary" gorm:"size:200;comment:辅助增量字段"` // 辅助增量字段（如 update_time），与主增量字段联合使用

	// 数据转换与过滤
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

	// Schema 演化感知
	SchemaEvolveMode SchemaEvolveMode `json:"schemaEvolveMode" gorm:"not null;default:0;comment:Schema演化检测 0关 1告警 2自动适配"` // Schema 变更检测模式

	// 双向同步
	BiDirEnabled        bool             `json:"biDirEnabled" gorm:"not null;default:false;comment:是否启用双向同步"`            // 是否启用双向同步
	ReverseTaskId       uint64           `json:"reverseTaskId" gorm:"comment:反向同步任务ID"`                                  // 反向同步任务 ID（0 表示未创建）
	ConflictStrategy    ConflictStrategy `json:"conflictStrategy" gorm:"not null;default:1;comment:冲突策略 1源优先 2目标优先 3跳过"` // 冲突仲裁策略
	BiDirTimestampField string           `json:"biDirTimestampField" gorm:"size:100;comment:双向同步时间戳比较字段"`                // 用于冲突检测的时间戳字段
}

func (d *DataSyncTask) TableName() string {
	return "t_db_data_sync_task"
}

// DataSyncLog 同步执行日志（含监控指标字段）
type DataSyncLog struct {
	model.IdModel

	CreateTime *time.Time `json:"createTime" gorm:"not null;"`            // 创建时间
	TaskId     uint64     `json:"taskId" gorm:"not null;comment:同步任务表id"` // 任务表id
	// RunId 本次执行的唯一标识（与 taskx.RunGuard 的持锁所有权值同值）。
	// 日志、启动收尾、水位推进、停止检查均围绕同一 runId 判定归属，避免“新实例启动误标旧实例日志为失败”与
	// “锁被接管后旧任务继续写入”。旧行可能为空（升级前无该列），启动收尾会当作 stale 处理。
	RunId       string `json:"runId" gorm:"size:64;index;comment:本次执行唯一标识"`                  // 本次执行唯一标识
	DataSQLFull string `json:"dataSqlFull" gorm:"not null;type:text;comment:执行的完整sql"`       // 执行的完整sql
	ResNum      int    `json:"resNum" gorm:"comment:收到数据条数"`                                 // 收到数据条数
	ErrText     string `json:"errText" gorm:"type:text;comment:日志"`                          // 日志
	Status      int8   `json:"status" gorm:"not null;default:1;comment:状态:1.成功 2.执行中 -1.失败"` // 状态:1.成功 2.执行中 -1.失败

	// 监控指标
	DurationMs    int64 `json:"durationMs" gorm:"comment:执行耗时(毫秒)"`     // 执行耗时（毫秒）
	Throughput    int   `json:"throughput" gorm:"comment:吞吐量(行/秒)"`     // 吞吐量（rows/sec）
	BatchCount    int   `json:"batchCount" gorm:"comment:批次数"`          // 执行批次数
	InsertCount   int   `json:"insertCount" gorm:"comment:新增行数"`        // INSERT 行数（含 UPSERT 中实际新增的部分）
	UpdateCount   int   `json:"updateCount" gorm:"comment:更新行数"`        // UPDATE 行数（UPSERT 中实际更新的部分）
	DeleteCount   int   `json:"deleteCount" gorm:"comment:删除行数"`        // DELETE 行数（硬删除模式）
	SkipCount     int   `json:"skipCount" gorm:"comment:跳过行数"`          // 跳过行数（过滤/冲突检测/空值/忽略策略未写入）
	SchemaChanges int   `json:"schemaChanges" gorm:"comment:Schema变更数"` // 本次检测到的 Schema 变更数

	// 运行日志：追加式执行过程记录（类似数据迁移的 AppendLog 机制）
	RunLog string `json:"runLog" gorm:"type:text;comment:运行日志"` // 运行日志（追加式）
}

func (d *DataSyncLog) TableName() string {
	return "t_db_data_sync_log"
}

// DataSyncTaskStatus 数据同步任务状态
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
