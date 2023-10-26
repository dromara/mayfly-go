package persistence

import (
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"strconv"
	"strings"
)

type machineRepoImpl struct {
	base.RepoImpl[*entity.Machine]
}

func newMachineRepo() repository.Machine {
	return &machineRepoImpl{base.RepoImpl[*entity.Machine]{M: new(entity.Machine)}}
}

// 分页获取机器信息列表
func (m *machineRepoImpl) GetMachineList(condition *entity.MachineQuery, pageParam *model.PageParam, toEntity *[]*vo.MachineVO, orderBy ...string) (*model.PageResult[*[]*vo.MachineVO], error) {
	qd := gormx.NewQuery(new(entity.Machine)).
		Like("ip", condition.Ip).
		Like("name", condition.Name).
		In("tag_id", condition.TagIds).
		RLike("tag_path", condition.TagPath).
		OrderByAsc("tag_path")

	if condition.Ids != "" {
		// ,分割id转为id数组
		qd.In("id", collx.ArrayMap[string, uint64](strings.Split(condition.Ids, ","), func(val string) uint64 {
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

	return m.CountByCond(where)
}
