package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/router"
)

func init() {
	initialize.AddInitIocFunc(application.InitIoc)
	initialize.AddInitRouterFunc(router.Init)
}
