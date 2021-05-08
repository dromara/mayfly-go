package persistence

import (
	"mayfly-go/base/model"
	"mayfly-go/devops/domain/entity"
	"mayfly-go/devops/domain/repository"
)

type accountRepo struct{}

var AccountDao repository.Account = &accountRepo{}

func (a *accountRepo) GetAccount(condition *entity.Account, cols ...string) error {
	return model.GetBy(condition, cols...)
}
