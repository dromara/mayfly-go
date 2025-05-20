package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type MachineCronJob interface {
	base.Repo[*entity.MachineCronJob]

	GetPageList(condition *entity.MachineCronJob, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineCronJob], error)
}

type MachineCronJobExec interface {
	base.Repo[*entity.MachineCronJobExec]

	GetPageList(condition *entity.MachineCronJobExec, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineCronJobExec], error)
}
