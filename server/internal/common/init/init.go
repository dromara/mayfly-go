package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/common/router"
)

func init() {
	initialize.AddInitRouterFunc(router.Init)
}
