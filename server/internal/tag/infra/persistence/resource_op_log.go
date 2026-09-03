package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
)

type resourceOpLogRepoImpl struct {
	base.RepoImpl[*entity.ResourceOpLog]
}

var _ repository.ResourceOpLog = (*resourceOpLogRepoImpl)(nil)

func newResourceOpLogRepo() repository.ResourceOpLog {
	return &resourceOpLogRepoImpl{}
}
