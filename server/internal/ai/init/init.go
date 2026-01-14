package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/ai/api"
)

func init() {
	// 注册AI模块的IoC组件
	initialize.AddInitIocFunc(func() {
		api.InitIoc()
	})
}
