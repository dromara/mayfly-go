package v112

import (
	dbentity "mayfly-go/internal/db/domain/entity"
	"mayfly-go/migration/migrations/migrationx"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// V1_12_4 v1.12.4 批次迁移。
//
// 批次号跟随应用版本（internal/pkg/config/app.go 的 Version），ID 前缀与批次版本一致。
// 本批次新增迁移统一收口在本文件，不按功能拆散；所有迁移自身幂等。
func v1124() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			// 文生 SQL 的业务词典表（语义层）。
			//
			// 表注释为空的存量库里，「活跃实例」这类业务口径在 schema 里完全不存在，
			// 词面召回与模型选择题都无从判断，实测会挑错同义表。词典把这类知识显式登记成
			// (资源, 库, 术语) → 表 + 口径说明，既喂召回也喂上下文。
			// 条目是无上界的集合且要按术语增删改，因此独立成表而不是塞进 Db.Extra
			// （Extra 是 varchar(2000) 且随库列表整体回显）。
			// AutoMigrate 对已有表按列比对补齐、对全新库直接建表，天然幂等。
			ID: "v1.12.4-db-ai-term",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&dbentity.DbAiTerm{})
			},
			Rollback: migrationx.NoopRollback,
		},
	}
}
