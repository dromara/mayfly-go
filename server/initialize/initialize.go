package initialize

import (
	dbInit "mayfly-go/internal/db/init"
	machineInit "mayfly-go/internal/machine/init"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
)

func InitOther() {
	// 为所有注册的实例注入其依赖的其他组件实例
	biz.ErrIsNil(ioc.DefaultContainer.InjectComponents())

	machineInit.Init()
	dbInit.Init()
}
