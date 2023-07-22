package repository

import "mayfly-go/internal/sys/domain/entity"

type OAuthAccount interface {
	// GetOAuthAccount 根据identity获取第三方账号信息
	GetOAuthAccount(condition *entity.OAuthAccount, cols ...string) error

	SaveOAuthAccount(e *entity.OAuthAccount) error
}
