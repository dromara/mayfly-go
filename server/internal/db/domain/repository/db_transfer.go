package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbTransferTask interface {
	base.Repo[*entity.DbTransferTask]

	// 分页获取数据库实例信息列表
	GetTaskList(condition *entity.DbTransferTaskQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferTask], error)
}

// DbTransferCheckpoint 迁移断点续传检查点仓储
type DbTransferCheckpoint interface {
	base.Repo[*entity.DbTransferCheckpoint]

	// GetByTaskId 获取指定任务的检查点，不存在返回nil
	GetByTaskId(taskId uint64) (*entity.DbTransferCheckpoint, error)
}
