package init

import (
	"mayfly-go/internal/auth/api"
	"mayfly-go/internal/auth/application"
	"mayfly-go/internal/auth/infra/persistence"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})
}
