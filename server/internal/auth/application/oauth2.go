package application

import (
	"context"
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/internal/auth/domain/repository"
	"mayfly-go/pkg/model"
)

type Oauth2 interface {
	GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error

	// BindOAuthAccount 绑定第三方身份。
	//
	// 必须带 ctx：绑定人就是这条记录的操作者，用无上下文的 ctx 会让创建人退化成系统账号，
	// 「谁把这个第三方账号绑到了自己账户上」事后无从查证
	BindOAuthAccount(ctx context.Context, e *entity.Oauth2Account) error

	// Unbind 解绑账号的全部第三方身份
	Unbind(ctx context.Context, accountId uint64) error
}

type oauth2AppImpl struct {
	oauth2AccountRepo repository.Oauth2Account `inject:"T"`
}

var _ Oauth2 = (*oauth2AppImpl)(nil)

func (a *oauth2AppImpl) GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error {
	return a.oauth2AccountRepo.GetByCond(model.NewModelCond(condition).Columns(cols...))
}

func (a *oauth2AppImpl) BindOAuthAccount(ctx context.Context, e *entity.Oauth2Account) error {
	if e.Id == 0 {
		return a.oauth2AccountRepo.Insert(ctx, e)
	}
	return a.oauth2AccountRepo.UpdateById(ctx, e)
}

func (a *oauth2AppImpl) Unbind(ctx context.Context, accountId uint64) error {
	return a.oauth2AccountRepo.DeleteByCond(ctx, &entity.Oauth2Account{AccountId: accountId})
}
