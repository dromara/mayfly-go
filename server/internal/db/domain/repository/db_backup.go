package repository

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/model"
)

type DbBackup interface {
	DbTask[*entity.DbBackup]

	// GetDbBackupList 分页获取数据信息列表
	GetDbBackupList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
	AddTask(ctx context.Context, tasks ...*entity.DbBackup) error
	GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error)
}
