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
	return &dbTransferTaskRepoImpl{}
}

// 分页获取数据库信息列表
func (d *dbTransferTaskRepoImpl) GetTaskList(condition *entity.DbTransferTaskQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferTask], error) {
	qd := model.NewCond().
		Like("task_name", condition.Name).
		Eq("status", condition.Status).
		Eq("cron_able", condition.CronAble)
	//Eq("status", condition.Status)
	return d.PageByCond(qd, condition.PageParam)
}
