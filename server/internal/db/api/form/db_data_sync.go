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
	CursorInclusivity     entity.CursorInclusivity `json:"cursorInclusivity"`
	SleepBetweenBatchesMs int                      `json:"sleepBetweenBatchesMs" binding:"omitempty,min=0,max=60000"`
	SkipIndexValidation   bool                     `json:"skipIndexValidation"`

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
