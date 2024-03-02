package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/auth/application"
	"mayfly-go/internal/auth/infrastructure/persistence"
	"mayfly-go/internal/auth/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
}
