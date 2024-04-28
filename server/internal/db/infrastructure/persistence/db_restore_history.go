package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

var _ repository.DbRestoreHistory = (*dbRestoreHistoryRepoImpl)(nil)

type dbRestoreHistoryRepoImpl struct {
	base.RepoImpl[*entity.DbRestoreHistory]
}

func NewDbRestoreHistoryRepo() repository.DbRestoreHistory {
	return &dbRestoreHistoryRepoImpl{}
}

func (d *dbRestoreHistoryRepoImpl) GetDbRestoreHistories(condition *entity.DbRestoreHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Eq("id", condition.Id).
		Eq("db_backup_id", condition.DbRestoreId)
	return d.PageByCond(qd, pageParam, toEntity)

}
