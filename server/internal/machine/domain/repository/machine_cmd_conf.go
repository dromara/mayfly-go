package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
)

type MachineCmdConf interface {
	base.Repo[*entity.MachineCmdConf]
}
