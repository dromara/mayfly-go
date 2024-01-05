package application

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DataSyncLog interface {
	base.App[*entity.DataSyncLog]

	// GetTaskLogList 分页获取数据库实例
	GetTaskLogList(condition *entity.DataSyncLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

func newDataSyncLogApp(dataSyncRepo repository.DataSyncLog) DataSyncLog {
	app := new(dataSyncLogAppImpl)
	app.Repo = dataSyncRepo
	return app
}

type dataSyncLogAppImpl struct {
	base.AppImpl[*entity.DataSyncLog, repository.DataSyncLog]
}

func (app *dataSyncLogAppImpl) GetTaskLogList(condition *entity.DataSyncLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.GetRepo().GetTaskLogList(condition, pageParam, toEntity, orderBy...)
}
