package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/redis/api"
	"mayfly-go/internal/redis/application"
	"mayfly-go/internal/redis/infra/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	initialize.AddInitFunc(application.Init)
}
