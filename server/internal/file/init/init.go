package init

import (
	"mayfly-go/internal/file/api"
	"mayfly-go/internal/file/application"
	"mayfly-go/internal/file/infra/persistence"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
