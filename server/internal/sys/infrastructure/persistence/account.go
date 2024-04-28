package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type AccountRepoImpl struct {
	base.RepoImpl[*entity.Account]
}

func newAccountRepo() repository.Account {
	return &AccountRepoImpl{base.RepoImpl[*entity.Account]{M: new(entity.Account)}}
}

func (m *AccountRepoImpl) GetPageList(condition *entity.AccountQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Like("username", condition.Username).
		In("id", condition.Ids)
	return m.PageByCond(qd, pageParam, toEntity)
}
