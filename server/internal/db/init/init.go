package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/infrastructure/persistence"
	"mayfly-go/internal/db/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})

	initialize.AddInitRouterFunc(router.Init)
	initialize.AddInitFunc(application.Init)
	initialize.AddTerminateFunc(Terminate)
}
