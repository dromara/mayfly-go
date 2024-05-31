package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
)

type machineCmdConfRepoImpl struct {
	base.RepoImpl[*entity.MachineCmdConf]
}

func newMachineCmdConfRepo() repository.MachineCmdConf {
	return &machineCmdConfRepoImpl{base.RepoImpl[*entity.MachineCmdConf]{M: new(entity.MachineCmdConf)}}
}
