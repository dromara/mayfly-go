package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
)

type resourceAuthCertRepoImpl struct {
	base.RepoImpl[*entity.ResourceAuthCert]
}

func newResourceAuthCertRepoImpl() repository.ResourceAuthCert {
	return &resourceAuthCertRepoImpl{}
}
