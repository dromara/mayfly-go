package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type machineRepoImpl struct {
	base.RepoImpl[*entity.Machine]
}

func newMachineRepo() repository.Machine {
	return &machineRepoImpl{base.RepoImpl[*entity.Machine]{M: new(entity.Machine)}}
}

// 分页获取机器信息列表
func (m *machineRepoImpl) GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Eq("status", condition.Status).
		Like("ip", condition.Ip).
		Like("name", condition.Name).
		In("code", condition.Codes).
		Like("code", condition.Code).
		In("id", condition.Ids)

	// 只查询ssh服务器
	if condition.Ssh == entity.MachineProtocolSsh {
		qd.Eq("protocol", entity.MachineProtocolSsh)
	}

	return m.PageByCond(qd, pageParam, toEntity)
}
