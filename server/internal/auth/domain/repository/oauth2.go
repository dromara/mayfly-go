package repository

import (
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/pkg/base"
)

type Oauth2Account interface {
	base.Repo[*entity.Oauth2Account]
}
