package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/auth/api"
	"mayfly-go/internal/auth/application"
	"mayfly-go/internal/auth/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
