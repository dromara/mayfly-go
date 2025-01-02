package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/file/api"
	"mayfly-go/internal/file/application"
	"mayfly-go/internal/file/infrastructure/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
