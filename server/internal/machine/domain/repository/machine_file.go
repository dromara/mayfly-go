package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type MachineFile interface {
	base.Repo[*entity.MachineFile]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MachineFile, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineFile], error)
}
