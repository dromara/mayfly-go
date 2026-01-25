package init

import (
	"mayfly-go/internal/flow/api"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/infra/persistence"
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
