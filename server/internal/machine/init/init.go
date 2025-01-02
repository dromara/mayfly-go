package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	initialize.AddInitFunc(application.Init)
}
