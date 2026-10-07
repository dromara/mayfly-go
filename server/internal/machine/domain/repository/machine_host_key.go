package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
)

type MachineHostKey interface {
	base.Repo[*entity.MachineHostKey]
}
