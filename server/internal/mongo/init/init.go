package init

import (
	"mayfly-go/internal/mongo/api"
	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/infra/persistence"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
