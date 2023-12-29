package application

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"
)

func newDbRestoreHistoryApp(repositories *repository.Repositories) (*DbRestoreHistoryApp, error) {
	app := &DbRestoreHistoryApp{
		repo: repositories.RestoreHistory,
	}
	return app, nil
}

type DbRestoreHistoryApp struct {
	repo repository.DbRestoreHistory
}

// GetPageList 分页获取数据库备份历史
func (app *DbRestoreHistoryApp) GetPageList(condition *entity.DbRestoreHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.repo.GetDbRestoreHistories(condition, pageParam, toEntity, orderBy...)
}
