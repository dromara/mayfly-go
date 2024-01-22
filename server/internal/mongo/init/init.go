package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/router"
)

func init() {
	initialize.AddInitIocFunc(application.InitIoc)
	initialize.AddInitRouterFunc(router.Init)
}
