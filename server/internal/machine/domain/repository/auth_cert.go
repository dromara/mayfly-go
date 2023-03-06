package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/model"
)

type AuthCert interface {
	GetPageList(condition *entity.AuthCert, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Insert(ac *entity.AuthCert)

	Update(ac *entity.AuthCert)

	GetById(id uint64) *entity.AuthCert

	GetByIds(ids ...uint64) []*entity.AuthCert

	GetByCondition(condition *entity.AuthCert, cols ...string) error

	DeleteById(id uint64)
}
