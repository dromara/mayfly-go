package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

// DbAiTerm 文生 SQL 业务词典仓储
type DbAiTerm interface {
	base.Repo[*entity.DbAiTerm]

	// ListByDb 取某个库资源下配置的全部词典条目（按术语升序，界面要稳定展示）。
	//
	// 不按 schema 过滤：一个资源下的条目数量级很小（词典是人工登记的），而「空 schema = 全库共用」
	// 这条语义留在应用层判定更清楚，仓储无需理解它
	ListByDb(dbId uint64) ([]*entity.DbAiTerm, error)
}
