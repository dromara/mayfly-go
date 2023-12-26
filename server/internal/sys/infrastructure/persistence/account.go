package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type accountRepoImpl struct {
	base.RepoImpl[*entity.Account]
}

func newAccountRepo() repository.Account {
	return &accountRepoImpl{base.RepoImpl[*entity.Account]{M: new(entity.Account)}}
}

func (m *accountRepoImpl) GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.Account)).
		Like("name", condition.Name).
		Like("username", condition.Username)
	return gormx.PageQuery(qd, pageParam, toEntity)
}
