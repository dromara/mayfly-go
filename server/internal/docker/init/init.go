package init

import (
	"mayfly-go/internal/docker/api"
	"mayfly-go/internal/docker/application"
	"mayfly-go/internal/docker/infra/persistence"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
