package persistence

import (
	"fmt"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
)

type machineRepo struct{}

var MachineDao repository.Machine = &machineRepo{}

// 分页获取机器信息列表
func (m *machineRepo) GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	sql := "SELECT m.* FROM t_machine m JOIN t_project_member pm ON m.project_id = pm.project_id WHERE 1 = 1 "
	if condition.CreatorId != 0 {
		// 使用创建者id模拟项目成员id
		sql = fmt.Sprintf("%s AND pm.account_id = %d", sql, condition.CreatorId)
	}
	if condition.ProjectId != 0 {
		sql = fmt.Sprintf("%s AND m.project_id = %d", sql, condition.ProjectId)
	}
	if condition.Ip != "" {
		sql = sql + " AND m.ip LIKE '%" + condition.Ip + "%'"
	}
	sql = sql + " ORDER BY m.create_time DESC"
	return model.GetPageBySql(sql, pageParam, toEntity)
}

func (m *machineRepo) Count(condition *entity.Machine) int64 {
	return model.CountBy(condition)
}

// 根据条件获取账号信息
func (m *machineRepo) GetMachine(condition *entity.Machine, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (m *machineRepo) GetById(id uint64, cols ...string) *entity.Machine {
	machine := new(entity.Machine)
	if err := model.GetById(machine, id, cols...); err != nil {
		return nil

	}
	return machine
}

func (m *machineRepo) Create(entity *entity.Machine) {
	model.Insert(entity)
}

func (m *machineRepo) UpdateById(entity *entity.Machine) {
	model.UpdateById(entity)
}
