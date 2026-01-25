package init

import (
	"mayfly-go/internal/db/ai/tools"
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/infra/persistence"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	starter.AddInitFunc(application.Init)
	starter.AddTerminateFunc(Terminate)
	// 注册AI数据库工具
	tools.Init()
}
