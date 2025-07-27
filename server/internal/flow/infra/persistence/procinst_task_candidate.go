package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
)

type procinstTaskCandidateImpl struct {
	base.RepoImpl[*entity.ProcinstTaskCandidate]
}

func newProcinstTaskCandidateRepo() repository.ProcinstTaskCandidate {
	return &procinstTaskCandidateImpl{}
}
