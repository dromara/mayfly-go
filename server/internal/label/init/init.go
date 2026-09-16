package init

import (
	"mayfly-go/internal/label/api"
	"mayfly-go/internal/label/application"
	"mayfly-go/internal/label/infra/persistence"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
