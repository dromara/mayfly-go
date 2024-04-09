package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
)

type ResourceAuthCert interface {
	base.Repo[*entity.ResourceAuthCert]
}
