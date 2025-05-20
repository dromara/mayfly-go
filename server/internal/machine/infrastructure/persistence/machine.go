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
	return &machineRepoImpl{}
}

// 分页获取机器信息列表
func (m *machineRepoImpl) GetMachineList(condition *entity.MachineQuery, orderBy ...string) (*model.PageResult[*entity.Machine], error) {
	qd := model.NewCond().
		Eq("id", condition.Id).
		Eq("status", condition.Status).
		Like("ip", condition.Ip).
		Like("name", condition.Name).
		In("code", condition.Codes).
		Eq("code", condition.Code).
		Eq("protocol", condition.Protocol)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("ip like ? or name like ? or code like ?", keyword, keyword, keyword)
	}

	return m.PageByCond(qd, condition.PageParam)
}
