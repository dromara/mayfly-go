package persistence

import (
	"mayfly-go/base/model"
	"mayfly-go/server/sys/domain/entity"
	"mayfly-go/server/sys/domain/repository"
)

type accountRepo struct{}

var AccountDao repository.Account = &accountRepo{}

func (a *accountRepo) GetAccount(condition *entity.Account, cols ...string) error {
	return model.GetBy(condition, cols...)
}

func (m *accountRepo) GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}
