package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/infrastructure/persistence"
	"mayfly-go/internal/sys/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
}
