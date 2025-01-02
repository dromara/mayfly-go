package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/mongo/api"
	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
