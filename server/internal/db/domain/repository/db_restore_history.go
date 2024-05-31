package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbRestoreHistory interface {
	base.Repo[*entity.DbRestoreHistory]

	// GetDbRestoreHistories 分页获取数据备份历史
	GetDbRestoreHistories(condition *entity.DbRestoreHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}
