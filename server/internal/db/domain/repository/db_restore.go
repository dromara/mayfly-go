package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbRestore interface {
	base.Repo[*entity.DbRestore]

	// GetDbRestoreList 分页获取数据信息列表
	GetDbRestoreList(condition *entity.DbRestoreQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
	AddTask(ctx context.Context, tasks ...*entity.DbRestore) error
	UpdateTaskStatus(ctx context.Context, task *entity.DbRestore) error
	GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error)
	UpdateEnabled(ctx context.Context, taskId uint64, enabled bool) error
}
