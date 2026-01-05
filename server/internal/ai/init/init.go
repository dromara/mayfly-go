package init

import (
	"mayfly-go/internal/ai/api"
	"mayfly-go/initialize"
)

func init() {
	// 注册AI模块的IoC组件
	initialize.AddInitIocFunc(func() {
		api.InitIoc()
	})
}
