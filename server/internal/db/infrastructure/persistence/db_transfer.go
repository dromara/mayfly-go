package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dbTransferTaskRepoImpl struct {
	base.RepoImpl[*entity.DbTransferTask]
}

func newDbTransferTaskRepo() repository.DbTransferTask {
	return &dbTransferTaskRepoImpl{base.RepoImpl[*entity.DbTransferTask]{M: new(entity.DbTransferTask)}}
}

// 分页获取数据库信息列表
func (d *dbTransferTaskRepoImpl) GetTaskList(condition *entity.DbTransferTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Like("task_name", condition.Name)
	//Eq("status", condition.Status)
	return d.PageByCondToAny(qd, pageParam, toEntity)
}
