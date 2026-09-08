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

var _ repository.DbTransferTask = (*dbTransferTaskRepoImpl)(nil)

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

type dbTransferCheckpointRepoImpl struct {
	base.RepoImpl[*entity.DbTransferCheckpoint]
}

var _ repository.DbTransferCheckpoint = (*dbTransferCheckpointRepoImpl)(nil)

func newDbTransferCheckpointRepo() repository.DbTransferCheckpoint {
	return &dbTransferCheckpointRepoImpl{}
}

// GetByTaskId 获取指定任务的检查点，不存在返回nil
func (d *dbTransferCheckpointRepoImpl) GetByTaskId(taskId uint64) (*entity.DbTransferCheckpoint, error) {
	list, err := d.SelectByCond(model.NewCond().Eq("task_id", taskId))
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}
