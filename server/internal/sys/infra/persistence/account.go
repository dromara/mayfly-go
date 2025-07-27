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
	return &AccountRepoImpl{}
}

func (m *AccountRepoImpl) GetPageList(condition *entity.AccountQuery, orderBy ...string) (*model.PageResult[*entity.Account], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Like("username", condition.Username).
		In("id", condition.Ids)
	return m.PageByCond(qd, condition.PageParam)
}
