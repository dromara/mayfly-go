package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/model"
)

type DbBackup interface {
	DbJob[*entity.DbBackup]

	ListToDo(jobs any) error
	ListDbInstances(enabled bool, repeated bool, instanceIds *[]uint64) error
	GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error)

	// GetPageList 分页获取数据库任务列表
	GetPageList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	ListByCond(cond any, listModels any, cols ...string) error
}
