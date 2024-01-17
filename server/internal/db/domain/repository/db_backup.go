package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/model"
)

type DbBackup interface {
	DbJob

	ListToDo(jobs any) error
	ListDbInstances(enabled bool, repeated bool, instanceIds *[]uint64) error
	GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error)

	// GetPageList 分页获取数据库任务列表
	GetPageList(condition *entity.DbJobQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}
