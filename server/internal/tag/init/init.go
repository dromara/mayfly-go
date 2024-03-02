package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/infrastructure/persistence"
	"mayfly-go/internal/tag/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
}
