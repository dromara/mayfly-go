package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/router"
)

func init() {
	initialize.AddInitIocFunc(application.InitIoc)
	initialize.AddInitRouterFunc(router.Init)
}
