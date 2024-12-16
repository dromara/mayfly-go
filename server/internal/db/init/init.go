package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	initialize.AddInitFunc(application.Init)
	initialize.AddTerminateFunc(Terminate)
}
