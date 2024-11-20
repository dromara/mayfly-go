package application

import (
	"context"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

type Account interface {
	base.App[*entity.Account]

	GetPageList(condition *entity.AccountQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Create(ctx context.Context, account *entity.Account) error

	Update(ctx context.Context, account *entity.Account) error

	Delete(ctx context.Context, id uint64) error
}

type accountAppImpl struct {
	base.AppImpl[*entity.Account, repository.Account]

	accountRoleRepo repository.AccountRole `inject:"AccountRoleRepo"`
}

// 注入AccountRepo
func (a *accountAppImpl) InjectAccountRepo(repo repository.Account) {
	a.Repo = repo
}

func (a *accountAppImpl) GetPageList(condition *entity.AccountQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return a.GetRepo().GetPageList(condition, pageParam, toEntity)
}

func (a *accountAppImpl) Create(ctx context.Context, account *entity.Account) error {
	if a.GetByCond(&entity.Account{Username: account.Username}) == nil {
		return errorx.NewBizI(ctx, imsg.ErrUsernameExist)
	}
	account.Status = entity.AccountEnable
	return a.Insert(ctx, account)
}

func (a *accountAppImpl) Update(ctx context.Context, account *entity.Account) error {
	if account.Username != "" {
		unAcc := &entity.Account{Username: account.Username}
		err := a.GetByCond(unAcc)
		if err == nil && unAcc.Id != account.Id {
			return errorx.NewBizI(ctx, imsg.ErrUsernameExist)
		}
	}

	return a.UpdateById(ctx, account)
}

func (a *accountAppImpl) Delete(ctx context.Context, id uint64) error {
	return a.Tx(ctx, func(ctx context.Context) error {
		return a.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		return a.accountRoleRepo.DeleteByCond(ctx, &entity.AccountRole{AccountId: id})
	})
}
