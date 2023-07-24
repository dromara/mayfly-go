package repository

import "mayfly-go/internal/auth/domain/entity"

type Oauth2Account interface {
	// GetOAuthAccount 根据identity获取第三方账号信息
	GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error

	SaveOAuthAccount(e *entity.Oauth2Account) error

	DeleteBy(e *entity.Oauth2Account)
}
