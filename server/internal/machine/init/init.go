package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/infrastructure/persistence"
	"mayfly-go/internal/machine/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
	initialize.AddInitFunc(application.Init)
}
