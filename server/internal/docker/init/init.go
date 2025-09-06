package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/docker/api"
	"mayfly-go/internal/docker/application"
	"mayfly-go/internal/docker/infra/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
