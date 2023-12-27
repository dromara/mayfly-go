package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbBackup interface {
	base.Repo[*entity.DbBackup]

	// GetDbBackupList 分页获取数据信息列表
	GetDbBackupList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
	AddTask(ctx context.Context, tasks ...*entity.DbBackup) error
	UpdateTaskStatus(ctx context.Context, task *entity.DbBackup) error
	GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error)
	UpdateEnabled(ctx context.Context, taskId uint64, enabled bool) error
}
