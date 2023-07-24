package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/model"
)

type Account interface {
	// 根据条件获取账号信息
	GetAccount(condition *entity.Account, cols ...string) error

	GetById(id uint64) *entity.Account

	GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Insert(account *entity.Account)

	Update(account *entity.Account)
}
