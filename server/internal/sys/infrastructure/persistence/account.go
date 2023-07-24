package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type accountRepoImpl struct{}

func newAccountRepo() repository.Account {
	return new(accountRepoImpl)
}

func (a *accountRepoImpl) GetAccount(condition *entity.Account, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

func (a *accountRepoImpl) GetById(id uint64) *entity.Account {
	ac := new(entity.Account)
	if err := gormx.GetById(ac, id); err != nil {
		return nil
	}
	return ac
}

func (m *accountRepoImpl) GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(new(entity.Account)).
		Like("name", condition.Name).
		Like("username", condition.Username)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (m *accountRepoImpl) Insert(account *entity.Account) {
	biz.ErrIsNil(gormx.Insert(account), "新增账号信息失败")
}

func (m *accountRepoImpl) Update(account *entity.Account) {
	biz.ErrIsNil(gormx.UpdateById(account), "更新账号信息失败")
}
