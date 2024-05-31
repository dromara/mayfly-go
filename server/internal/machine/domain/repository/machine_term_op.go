package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type MachineTermOp interface {
	base.Repo[*entity.MachineTermOp]

	// 分页获取机器终端执行记录列表
	GetPageList(condition *entity.MachineTermOp, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 根据条件获取记录列表
	SelectByQuery(cond *entity.MachineTermOpQuery) ([]*entity.MachineTermOp, error)
}
