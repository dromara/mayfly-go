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
	DataSql     string `binding:"required" json:"dataSql"`
	PageSize    int    `binding:"required" json:"pageSize"`
	UpdField    string `binding:"required" json:"updField"`
	UpdFieldVal string `binding:"required" json:"updFieldVal"`
	UpdFieldSrc string `json:"updFieldSrc"`

	// 多字段增量条件
	UpdFieldSecondary string `json:"updFieldSecondary"`

	// 目标数据库信息
	TargetDbId        int64  `binding:"required" json:"targetDbId"`
	TargetDbName      string `binding:"required" json:"targetDbName"`
	TargetTagPath     string `binding:"required" json:"targetTagPath"`
	TargetTableName   string `binding:"required" json:"targetTableName"`
	FieldMap          string `binding:"required" json:"fieldMap"`
	DuplicateStrategy int    `json:"duplicateStrategy"`

	// Phase 3: 数据转换与过滤
	TransformRules  string              `json:"transformRules"`
	FilterCondition string              `json:"filterCondition"`
	NullStrategy    entity.NullStrategy `json:"nullStrategy"`
	NullDefault     string              `json:"nullDefault"`

	// 删除同步配置（SyncMode 为 4/5 时使用）
	SoftDeleteField string `json:"softDeleteField"`
	SoftDeleteValue string `json:"softDeleteValue"`

	// Phase 5: Schema 演化感知
	SchemaEvolveMode entity.SchemaEvolveMode `json:"schemaEvolveMode"`

	// Phase 6.3: 双向同步
	BiDirEnabled        bool                    `json:"biDirEnabled"`
	ConflictStrategy    entity.ConflictStrategy `json:"conflictStrategy"`
	BiDirTimestampField string                  `json:"biDirTimestampField"`
}

type DataSyncTaskStatusForm struct {
	Id     uint64 `binding:"required" json:"taskId"`
	Status int8   `json:"status"`
}
