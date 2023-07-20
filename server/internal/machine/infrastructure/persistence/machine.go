package persistence

import (
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"strconv"
	"strings"
)

type machineRepoImpl struct{}

func newMachineRepo() repository.Machine {
	return new(machineRepoImpl)
}

// 分页获取机器信息列表
func (m *machineRepoImpl) GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity *[]*vo.MachineVO, orderBy ...string) *model.PageResult[*[]*vo.MachineVO] {
	qd := gormx.NewQuery(new(entity.Machine)).
		Like("ip", condition.Ip).
		Like("name", condition.Name).
		In("tag_id", condition.TagIds).
		RLike("tag_path", condition.TagPath).
		OrderByAsc("tag_path")

	if condition.Ids != "" {
		// ,分割id转为id数组
		qd.In("id", utils.ArrayMap[string, uint64](strings.Split(condition.Ids, ","), func(val string) uint64 {
			id, _ := strconv.Atoi(val)
			return uint64(id)
		}))
	}

	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (m *machineRepoImpl) Count(condition *entity.MachineQuery) int64 {
	where := make(map[string]any)
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}

	return gormx.CountByCond(new(entity.Machine), where)
}

// 根据条件获取账号信息
func (m *machineRepoImpl) GetMachine(condition *entity.Machine, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

// 根据id获取
func (m *machineRepoImpl) GetById(id uint64, cols ...string) *entity.Machine {
	machine := new(entity.Machine)
	if err := gormx.GetById(machine, id, cols...); err != nil {
		return nil

	}
	return machine
}

func (m *machineRepoImpl) Create(entity *entity.Machine) {
	biz.ErrIsNilAppendErr(gormx.Insert(entity), "创建机器信息失败: %s")
}

func (m *machineRepoImpl) UpdateById(entity *entity.Machine) {
	biz.ErrIsNilAppendErr(gormx.UpdateById(entity), "更新机器信息失败: %s")
}
