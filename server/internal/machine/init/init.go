package init

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/infra/persistence"
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
