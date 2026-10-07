package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
)

type machineHostKeyRepoImpl struct {
	base.RepoImpl[*entity.MachineHostKey]
}

var _ repository.MachineHostKey = (*machineHostKeyRepoImpl)(nil)

func newMachineHostKeyRepo() repository.MachineHostKey {
	return &machineHostKeyRepoImpl{}
}
