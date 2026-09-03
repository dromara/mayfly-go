package persistence

import (
	"mayfly-go/internal/file/domain/entity"
	"mayfly-go/internal/file/domain/repository"
	"mayfly-go/pkg/base"
)

type fileRepoImpl struct {
	base.RepoImpl[*entity.File]
}

var _ repository.File = (*fileRepoImpl)(nil)

func newFileRepo() repository.File {
	return &fileRepoImpl{}
}
