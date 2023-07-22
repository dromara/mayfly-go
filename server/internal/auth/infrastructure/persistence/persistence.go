package persistence

import "mayfly-go/internal/auth/domain/repository"

var (
	authAccountRepo = newAuthAccountRepo()
)

func GetOauthAccountRepo() repository.Oauth2Account {
	return authAccountRepo
}
