package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/infrastructure/persistence"
	"mayfly-go/internal/mongo/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
}
