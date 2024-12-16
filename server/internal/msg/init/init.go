package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/msg/api"
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
