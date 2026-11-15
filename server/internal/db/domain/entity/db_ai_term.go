package entity

import (
	"mayfly-go/pkg/model"
)

// DbAiTerm 文生 SQL 的业务词典条目（语义层最小形态）。
//
// 为什么需要它：确定性召回只有两条词面通道（表名、表注释），而大量存量库的表注释为空，
// 「活跃实例」「坏账率」这类业务口径在 schema 里根本不存在，模型只能凭表名猜——真机实测
// 就会挑错同义表。业界同类产品（语义模型 / MDL）都是把这类知识**显式登记**而不是让模型猜，
// 本实体就是那份登记。
//
// 为什么独立成表而不是塞进 Db 的 Extra：词典是按库/按术语增删改的集合，条数无上界，
// Extra 是 varchar(2000) 且随库列表整体回显；与「表名清单独立成表而非进 Extra」同判据。
//
// Schema 为空串表示该资源（库连接）下的所有库共用，便于全库型资源一次登记。
type DbAiTerm struct {
	model.Model

	DbId uint64 `json:"dbId" gorm:"not null;index:idx_db_ai_term_db;comment:数据库资源id"`
	// 列名用 schema_name：schema 在部分方言里是关键字，直接当列名会建表失败
	Schema  string   `json:"schema" gorm:"column:schema_name;size:150;comment:库名，空=该资源下所有库共用"`
	Term    string   `json:"term" gorm:"size:150;not null;comment:业务术语/指标名"`
	Tables  []string `json:"tables" gorm:"column:table_names;type:json;serializer:json;comment:该术语对应的表名"`
	Details string   `json:"details" gorm:"size:1000;comment:口径说明（取值含义、计算规则、默认过滤条件）"`
}
