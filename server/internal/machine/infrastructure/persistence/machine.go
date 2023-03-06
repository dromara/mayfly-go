package persistence

import (
	"fmt"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type machineRepoImpl struct{}

func newMachineRepo() repository.Machine {
	return new(machineRepoImpl)
}

// 分页获取机器信息列表
func (m *machineRepoImpl) GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	sql := "SELECT m.* FROM t_machine m WHERE 1 = 1 "

	values := make([]interface{}, 0)
	if condition.Ip != "" {
		sql = sql + " AND m.ip LIKE ?"
		values = append(values, "%"+condition.Ip+"%")
	}
	if condition.Name != "" {
		sql = sql + " AND m.name LIKE ?"
		values = append(values, "%"+condition.Name+"%")
	}
	if len(condition.TagIds) > 0 {
		sql = fmt.Sprintf("%s AND m.tag_id IN ? ", sql)
		values = append(values, condition.TagIds)
	}
	if condition.TagPathLike != "" {
		sql = sql + " AND m.tag_path LIKE ?"
		values = append(values, condition.TagPathLike+"%")
	}
	sql = sql + " ORDER BY m.tag_path"
	return model.GetPageBySql(sql, pageParam, toEntity, values...)
}

func (m *machineRepoImpl) Count(condition *entity.MachineQuery) int64 {
	where := make(map[string]interface{})
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}
	if condition.TagId != 0 {
		where["tag_id"] = condition.TagId
	}

	return model.CountByMap(new(entity.Machine), where)
}

// 根据条件获取账号信息
func (m *machineRepoImpl) GetMachine(condition *entity.Machine, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (m *machineRepoImpl) GetById(id uint64, cols ...string) *entity.Machine {
	machine := new(entity.Machine)
	if err := model.GetById(machine, id, cols...); err != nil {
		return nil

	}
	return machine
}

func (m *machineRepoImpl) Create(entity *entity.Machine) {
	biz.ErrIsNilAppendErr(model.Insert(entity), "创建机器信息失败: %s")
}

func (m *machineRepoImpl) UpdateById(entity *entity.Machine) {
	biz.ErrIsNilAppendErr(model.UpdateById(entity), "更新机器信息失败: %s")
}
