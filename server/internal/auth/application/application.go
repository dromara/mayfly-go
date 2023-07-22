package application

import "mayfly-go/internal/auth/infrastructure/persistence"

var (
	authApp = newAuthApp(persistence.GetOauthAccountRepo())
)

func GetAuthApp() Oauth2 {
	return authApp
}
