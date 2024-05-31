package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/redis/application"
	"mayfly-go/internal/redis/infrastructure/persistence"
	"mayfly-go/internal/redis/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})

	initialize.AddInitRouterFunc(router.Init)
	initialize.AddInitFunc(application.Init)
}
