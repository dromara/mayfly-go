package application

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"
)

func newDbBackupHistoryApp(repositories *repository.Repositories) (*DbBackupHistoryApp, error) {
	app := &DbBackupHistoryApp{
		repo: repositories.BackupHistory,
	}
	return app, nil
}

type DbBackupHistoryApp struct {
	repo repository.DbBackupHistory
}

// GetPageList 分页获取数据库备份历史
func (app *DbBackupHistoryApp) GetPageList(condition *entity.DbBackupHistoryQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.repo.GetHistories(condition, pageParam, toEntity, orderBy...)
}
