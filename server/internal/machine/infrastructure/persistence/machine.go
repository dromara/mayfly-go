package persistence

import (
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
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
		Eq("status", condition.Status).
		Like("ip", condition.Ip).
		Like("name", condition.Name).
		In("code", condition.Codes)

	// 只查询ssh服务器
	if condition.Ssh == entity.MachineProtocolSsh {
		qd.Eq("protocol", entity.MachineProtocolSsh)
	}

	if condition.Ids != "" {
		// ,分割id转为id数组
		qd.In("id", collx.ArrayMap[string, uint64](strings.Split(condition.Ids, ","), func(val string) uint64 {
			return cast.ToUint64(val)
		}))
	}

	return gormx.PageQuery(qd, pageParam, toEntity)
}
