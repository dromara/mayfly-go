package application

import (
	"mayfly-go/devops/domain/entity"
	"mayfly-go/devops/domain/repository"
	"mayfly-go/devops/infrastructure/persistence"
)

type IAccount interface {
	GetAccount(condition *entity.Account, cols ...string) error
}

type accountApp struct {
	accountRepo repository.Account
}

var Account IAccount = &accountApp{accountRepo: persistence.AccountDao}

// 根据条件获取账号信息
func (a *accountApp) GetAccount(condition *entity.Account, cols ...string) error {
	return a.accountRepo.GetAccount(condition, cols...)
}
