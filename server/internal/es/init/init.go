package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/es/api"
	"mayfly-go/internal/es/application"
	"mayfly-go/internal/es/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	initialize.AddInitFunc(application.Init)
}
