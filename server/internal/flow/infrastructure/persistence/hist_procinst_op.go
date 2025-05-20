package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
)

type hisProcinstOpImpl struct {
	base.RepoImpl[*entity.HisProcinstOp]
}

func newHisProcinstOpRepo() repository.HisProcinstOp {
	return &hisProcinstOpImpl{}
}
