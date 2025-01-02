package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
)

type resourceOpLogRepoImpl struct {
	base.RepoImpl[*entity.ResourceOpLog]
}

func newResourceOpLogRepo() repository.ResourceOpLog {
	return &resourceOpLogRepoImpl{}
}
