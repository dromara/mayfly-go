package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/file/application"
	"mayfly-go/internal/file/infrastructure/persistence"
	"mayfly-go/internal/file/router"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
}
