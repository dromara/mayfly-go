package application

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	fileapp "mayfly-go/internal/file/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbTransferFile interface {
	base.App[*entity.DbTransferFile]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DbTransferFileQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferFile], error)

	Save(ctx context.Context, instanceEntity *entity.DbTransferFile) error

	Delete(ctx context.Context, id ...uint64) error
}

var _ DbTransferFile = (*dbTransferFileAppImpl)(nil)

type dbTransferFileAppImpl struct {
	base.AppImpl[*entity.DbTransferFile, repository.DbTransferFile]

	fileApp fileapp.File `inject:"T"`
}

func (app *dbTransferFileAppImpl) GetPageList(condition *entity.DbTransferFileQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferFile], error) {
	return app.GetRepo().GetPageList(condition, orderBy...)
}

func (app *dbTransferFileAppImpl) Save(ctx context.Context, taskEntity *entity.DbTransferFile) error {
	var err error
	if taskEntity.Id == 0 {
		err = app.Insert(ctx, taskEntity)
	} else {
		err = app.UpdateById(ctx, taskEntity)
	}
	return err
}

func (app *dbTransferFileAppImpl) Delete(ctx context.Context, id ...uint64) error {
	arr, err := app.GetByIds(id, "task_id", "file_key")
	if err != nil {
		return err
	}

	// 删除对应的文件
	for _, file := range arr {
		app.fileApp.Remove(ctx, file.FileKey)
	}

	// 删除数据
	return app.DeleteById(ctx, id...)
}
