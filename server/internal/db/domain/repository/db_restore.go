package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/model"
)

type DbRestore interface {
	DbJob[*entity.DbRestore]

	ListToDo(jobs any) error
	GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error)

	// GetPageList 分页获取数据库任务列表
	GetPageList(condition *entity.DbRestoreQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	GetEnabledRestores(toEntity any, backupHistoryId ...uint64) error
}
