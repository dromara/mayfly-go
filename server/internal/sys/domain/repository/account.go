package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Account interface {
	base.Repo[*entity.Account]

	GetPageList(condition *entity.AccountQuery, orderBy ...string) (*model.PageResult[*entity.Account], error)
}
