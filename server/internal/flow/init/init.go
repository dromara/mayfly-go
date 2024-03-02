package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/infrastructure/persistence"
	"mayfly-go/internal/flow/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
}
