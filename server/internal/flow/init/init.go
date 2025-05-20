package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/flow/api"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	initialize.AddInitFunc(application.Init)
}
