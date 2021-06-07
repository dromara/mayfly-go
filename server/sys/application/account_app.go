package application

import (
	"mayfly-go/base/model"
	"mayfly-go/server/sys/domain/entity"
	"mayfly-go/server/sys/domain/repository"
	"mayfly-go/server/sys/infrastructure/persistence"
)

type IAccount interface {
	GetAccount(condition *entity.Account, cols ...string) error

	GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult
}

type accountApp struct {
	accountRepo repository.Account
}

var Account IAccount = &accountApp{accountRepo: persistence.AccountDao}

// 根据条件获取账号信息
func (a *accountApp) GetAccount(condition *entity.Account, cols ...string) error {
	return a.accountRepo.GetAccount(condition, cols...)
}

func (a *accountApp) GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return a.accountRepo.GetPageList(condition, pageParam, toEntity)
}
