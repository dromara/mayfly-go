package init

import (
	"mayfly-go/internal/es/api"
	"mayfly-go/internal/es/application"
	"mayfly-go/internal/es/infra/persistence"
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
