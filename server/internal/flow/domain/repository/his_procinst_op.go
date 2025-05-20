package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
)

type HisProcinstOp interface {
	base.Repo[*entity.HisProcinstOp]
}
