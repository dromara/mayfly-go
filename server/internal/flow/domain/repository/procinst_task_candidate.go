package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
)

type ProcinstTaskCandidate interface {
	base.Repo[*entity.ProcinstTaskCandidate]
}
