package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/redis/application"
	"mayfly-go/internal/redis/router"
)

func init() {
	initialize.AddInitIocFunc(application.InitIoc)
	initialize.AddInitRouterFunc(router.Init)
}
