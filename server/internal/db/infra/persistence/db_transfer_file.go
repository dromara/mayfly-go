package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dbTransferFileRepoImpl struct {
	base.RepoImpl[*entity.DbTransferFile]
}

var _ repository.DbTransferFile = (*dbTransferFileRepoImpl)(nil)

func newDbTransferFileRepo() repository.DbTransferFile {
	return &dbTransferFileRepoImpl{}
}

// 分页获取迁移文件列表
func (d *dbTransferFileRepoImpl) GetPageList(condition *entity.DbTransferFileQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferFile], error) {
	qd := model.NewCond().
		Eq("task_id", condition.TaskId).
		OrderByDesc("create_time")
	return d.PageByCond(qd, condition.PageParam)
}
