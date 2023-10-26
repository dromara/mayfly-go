package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type AuthCert interface {
	base.Repo[*entity.AuthCert]

	GetPageList(condition *entity.AuthCertQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// GetByIds(ids ...uint64) []*entity.AuthCert
}
