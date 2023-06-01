package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type accountRepoImpl struct{}

func newAccountRepo() repository.Account {
	return new(accountRepoImpl)
}

func (a *accountRepoImpl) GetAccount(condition *entity.Account, cols ...string) error {
	return model.GetBy(condition, cols...)
}

func (m *accountRepoImpl) GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult {
	sql := "SELECT * FROM t_sys_account "
	username := condition.Username
	values := make([]any, 0)
	if username != "" {
		sql = sql + " WHERE username LIKE ?"
		values = append(values, "%"+username+"%")
	}
	return model.GetPageBySql(sql, pageParam, toEntity, values...)
}

func (m *accountRepoImpl) Insert(account *entity.Account) {
	biz.ErrIsNil(model.Insert(account), "新增账号信息失败")
}

func (m *accountRepoImpl) Update(account *entity.Account) {
	biz.ErrIsNil(model.UpdateById(account), "更新账号信息失败")
}
