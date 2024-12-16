package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/redis/api"
	"mayfly-go/internal/redis/application"
	"mayfly-go/internal/redis/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	initialize.AddInitFunc(application.Init)
}
