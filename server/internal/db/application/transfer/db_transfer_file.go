package transfer

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	fileapp "mayfly-go/internal/file/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

type DbTransferFile interface {
	base.App[*entity.DbTransferFile]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DbTransferFileQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferFile], error)

	Save(ctx context.Context, instanceEntity *entity.DbTransferFile) error

	Delete(ctx context.Context, id ...uint64) error
}

var _ DbTransferFile = (*DbTransferFileAppImpl)(nil)

type DbTransferFileAppImpl struct {
	base.AppImpl[*entity.DbTransferFile, repository.DbTransferFile]

	fileApp fileapp.File `inject:"T"`
}

func (app *DbTransferFileAppImpl) GetPageList(condition *entity.DbTransferFileQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferFile], error) {
	return app.GetRepo().GetPageList(condition, orderBy...)
}

func (app *DbTransferFileAppImpl) Save(ctx context.Context, taskEntity *entity.DbTransferFile) error {
	var err error
	if taskEntity.Id == 0 {
		err = app.Insert(ctx, taskEntity)
	} else {
		err = app.UpdateById(ctx, taskEntity)
	}
	return err
}

func (app *DbTransferFileAppImpl) Delete(ctx context.Context, id ...uint64) error {
	arr, err := app.GetByIds(id, "task_id", "file_key")
	if err != nil {
		return err
	}

	// 删除对应的文件：文件可能已被存储侧生命周期策略清理，此处记录失败原因但不阻断任务记录删除，
	// 否则一个失文的file_key会让该记录永远无法删除（旧实现直接丢弃返回值，文件残留无人知晓）
	for _, file := range arr {
		if file.FileKey == "" {
			continue
		}
		if err := app.fileApp.Remove(ctx, file.FileKey); err != nil {
			logx.WarnfContext(ctx, "failed to remove the db transfer file [%s]: %s", file.FileKey, err.Error())
		}
	}

	// 删除数据
	return app.DeleteById(ctx, id...)
}
