package application

import (
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/internal/auth/domain/repository"
	"mayfly-go/pkg/biz"

	"gorm.io/gorm"
)

type Oauth2 interface {
	GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error

	BindOAuthAccount(e *entity.Oauth2Account) error

	Unbind(accountId uint64)
}

func newAuthApp(oauthAccountRepo repository.Oauth2Account) Oauth2 {
	return &oauth2AppImpl{
		oauthAccountRepo: oauthAccountRepo,
	}
}

type oauth2AppImpl struct {
	oauthAccountRepo repository.Oauth2Account
}

func (a *oauth2AppImpl) GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error {
	err := a.oauthAccountRepo.GetOAuthAccount(condition, cols...)
	if err != nil {
		// 排除其他报错，如表不存在等
		biz.IsTrue(err == gorm.ErrRecordNotFound, "查询失败: %s", err.Error())
	}
	return err
}

func (a *oauth2AppImpl) BindOAuthAccount(e *entity.Oauth2Account) error {
	return a.oauthAccountRepo.SaveOAuthAccount(e)
}

func (a *oauth2AppImpl) Unbind(accountId uint64) {
	a.oauthAccountRepo.DeleteBy(&entity.Oauth2Account{AccountId: accountId})
}
