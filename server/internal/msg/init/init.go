package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/infrastructure/persistence"
	"mayfly-go/internal/msg/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
}
