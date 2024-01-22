package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/router"
)

func init() {
	initialize.AddInitIocFunc(application.InitIoc)
	initialize.AddInitRouterFunc(router.Init)
}
