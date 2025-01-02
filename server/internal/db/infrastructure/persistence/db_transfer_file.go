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

func newDbTransferFileRepo() repository.DbTransferFile {
	return &dbTransferFileRepoImpl{}
}

// 分页获取数据库信息列表
func (d *dbTransferFileRepoImpl) GetPageList(condition *entity.DbTransferFileQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Eq("task_id", condition.TaskId).
		OrderByDesc("create_time")
	//Eq("status", condition.Status)
	return d.PageByCondToAny(qd, pageParam, toEntity)
}
