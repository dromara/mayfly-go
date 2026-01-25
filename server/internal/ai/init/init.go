package init

import (
	"mayfly-go/internal/ai/api"
	"mayfly-go/pkg/starter"
)

func init() {
	// 注册AI模块的IoC组件
	starter.AddInitIocFunc(func() {
		api.InitIoc()
	})
}
