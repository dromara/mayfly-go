package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/db/ai/tools"
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/infra/persistence"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	initialize.AddInitFunc(application.Init)
	initialize.AddTerminateFunc(Terminate)
	// 注册AI数据库工具
	tools.Init()
}
