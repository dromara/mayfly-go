package init

import (
	"mayfly-go/internal/redis/api"
	"mayfly-go/internal/redis/application"
	"mayfly-go/internal/redis/infra/persistence"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	starter.AddInitFunc(application.Init)
}
