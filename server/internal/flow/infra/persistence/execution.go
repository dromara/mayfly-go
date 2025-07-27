package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
)

type exectionImpl struct {
	base.RepoImpl[*entity.Execution]
}

func newExectionRepo() repository.Execution {
	return &exectionImpl{}
}
