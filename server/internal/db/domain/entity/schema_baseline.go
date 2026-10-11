package entity

import (
	"time"

	"mayfly-go/pkg/model"
)

// DbSchemaBaseline 表结构基线（一张表一行）。
//
// 拆到表粒度而不是整库一行 JSON 的理由：
//   - 结构快照序列化后轻松上百 KB，塞进一个 varchar 列会在 mysql 上撞行长度上限，
//     截断存储又会让「漂移明细」悄悄失真；
//   - 表粒度天然给出「哪张表变了」这一最有用的信息，也无需再设计一份明细格式。
//
// 基线指纹（Fingerprint）代表「用户确认过的结构」，只有显式刷新基线才会变；
// Drifted/DriftDetail 是最近一次比对的结果，由定时任务写入，两者生命周期不同，不能合并
type DbSchemaBaseline struct {
	model.Model

	DbId uint64 `json:"dbId" gorm:"not null;index:idx_schema_baseline_db"`
	// InstanceId 冗余存一份所属实例 id：告警按实例评估，否则每评一次都要关联查一遍 t_db
	InstanceId uint64 `json:"instanceId" gorm:"not null;index:idx_schema_baseline_inst"`
	// Db 库名（pg 系可能是 `库/schema` 形态，与结构对比的库标识同源）
	Db string `json:"db" gorm:"size:150;not null;"`
	// Table 表名，一律以小写存：比对按表名不区分大小写进行
	Table string `json:"table" gorm:"size:150;not null;"`

	// Fingerprint 基线结构指纹；空串表示这张表是漂移时才被记下的（基线里没有它）
	Fingerprint string `json:"fingerprint" gorm:"size:64;"`
	// ColumnCount/PrimaryKeys/IndexCount 指纹之外的人类可读摘要，
	// 让告警能说「列数 8→9」而不只是「结构变了」
	ColumnCount int    `json:"columnCount" gorm:"not null;"`
	PrimaryKeys string `json:"primaryKeys" gorm:"size:500;"`
	IndexCount  int    `json:"indexCount" gorm:"not null;"`

	// Drifted 最近一次比对是否发现漂移
	Drifted bool `json:"drifted" gorm:"not null;index:idx_schema_baseline_drift"`
	// DriftDetail 漂移摘要（如「列数 8→9；主键 id→id,tenant_id」）
	DriftDetail string `json:"driftDetail" gorm:"size:1000;"`
	// CheckedAt 最近一次比对时间；从未比对过时为 nil（不能拿零值当「1970 年检查过」）
	CheckedAt *time.Time `json:"checkedAt"`
}

func (d *DbSchemaBaseline) TableName() string {
	return "t_db_schema_baseline"
}
