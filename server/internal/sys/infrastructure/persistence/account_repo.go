package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type accountRepo struct{}

var AccountDao repository.Account = &accountRepo{}

func (a *accountRepo) GetAccount(condition *entity.Account, cols ...string) error {
	return model.GetBy(condition, cols...)
}

func (m *accountRepo) GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	sql := "SELECT * FROM t_sys_account "
	username := condition.Username
	if username != "" {
		sql = sql + " WHERE username LIKE '%" + username + "%'"
	}
	return model.GetPageBySql(sql, pageParam, toEntity)
	// return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

func (m *accountRepo) Insert(account *entity.Account) {
	biz.ErrIsNil(model.Insert(account), "新增账号信息失败")
}

func (m *accountRepo) Update(account *entity.Account) {
	biz.ErrIsNil(model.UpdateById(account), "更新账号信息失败")
}
