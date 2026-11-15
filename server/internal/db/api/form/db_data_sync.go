package form

import "mayfly-go/internal/db/domain/entity"

type DataSyncTaskForm struct {
	Id       uint64 `json:"id"`
	TaskName string `binding:"required" json:"taskName"`
	TaskCron string `binding:"required" json:"taskCron"`
	Status   int    `binding:"required" json:"status"`

	// 同步模式
	SyncMode entity.DataSyncMode `json:"syncMode"`

	// 源数据库信息
	SrcDbId     int64  `binding:"required" json:"srcDbId"`
	SrcDbName   string `binding:"required" json:"srcDbName"`
	SrcTagPath  string `binding:"required" json:"srcTagPath"`
	DataSQL     string `binding:"required" json:"dataSql"`
	PageSize    int    `binding:"required" json:"pageSize"`
	UpdField    string `binding:"required" json:"updField"`
	UpdFieldVal string `binding:"required" json:"updFieldVal"`
	UpdFieldSrc string `json:"updFieldSrc"`

	// 多字段增量条件
	UpdFieldSecondary string `json:"updFieldSecondary"`

	// 交付语义与节流（P0）
	// CursorInclusivity：0=Auto、1=Exclusive(>)、2=Inclusive(>=，需目标 UPSERT 幂等)；Save 侧校验与模式组合合法性
	// SleepBetweenBatchesMs：0-60000，批间等待毫秒（目标侧节流）；上限在 Save 处校验
	// SkipIndexValidation：P1 逃生阀，视图/函数索引/无法推断单表时手动开启后跳过增量字段索引探测（存于 Extra）
	// ShardTargetRows：源侧主键分片扫描的单片目标行数，0=自动（小表单片等于不分片）；
	//   越界值在 Save 拒（运行期会钳制，但静默钳掉显式配置就是「配了但无效果」）；
	//   binding 拦住负值，否则 -1 与 0（自动）写入的 Extra 完全一致，同一个字段两种失败模式
	CursorInclusivity     entity.CursorInclusivity `json:"cursorInclusivity"`
	SleepBetweenBatchesMs int                      `json:"sleepBetweenBatchesMs" binding:"omitempty,min=0,max=60000"`
	SkipIndexValidation   bool                     `json:"skipIndexValidation"`
	ShardTargetRows       int                      `json:"shardTargetRows" binding:"omitempty,min=0"`
	// CountWriteSplit：写入前探测目标冲突键以拆出「新增/更新」计数（存于 Extra）。
	// 用指针区分「未传（默认开启）」与「显式关闭」——用 bool 会让不传该字段的调用方
	// （存量前端、API 直连、脚本）把历史口径静默改成关闭。
	CountWriteSplit *bool `json:"countWriteSplit"`

	// 目标数据库信息
	TargetDbId        int64  `binding:"required" json:"targetDbId"`
	TargetDbName      string `binding:"required" json:"targetDbName"`
	TargetTagPath     string `binding:"required" json:"targetTagPath"`
	TargetTableName   string `binding:"required" json:"targetTableName"`
	FieldMap          string `binding:"required" json:"fieldMap"`
	DuplicateStrategy int    `json:"duplicateStrategy"`

	// 数据转换与过滤
	TransformRules  string              `json:"transformRules"`
	FilterCondition string              `json:"filterCondition"`
	NullStrategy    entity.NullStrategy `json:"nullStrategy"`
	NullDefault     string              `json:"nullDefault"`

	// 删除同步配置（SyncMode 为 4/5 时使用）
	SoftDeleteField string `json:"softDeleteField"`
	SoftDeleteValue string `json:"softDeleteValue"`

	// Schema 演化感知
	SchemaEvolveMode entity.SchemaEvolveMode `json:"schemaEvolveMode"`

	// 双向同步
	BiDirEnabled        bool                    `json:"biDirEnabled"`
	ConflictStrategy    entity.ConflictStrategy `json:"conflictStrategy"`
	BiDirTimestampField string                  `json:"biDirTimestampField"`
}

type DataSyncTaskStatusForm struct {
	Id     uint64 `binding:"required" json:"taskId"`
	Status int8   `json:"status"`
}
