package initialize

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
)

// 初始化ioc函数
type InitIocFunc func()

// 初始化函数
type InitFunc func()

var (
	initIocFuncs = make([]InitIocFunc, 0)
	initFuncs    = make([]InitFunc, 0)
)

// 添加初始化ioc函数，由各个默认自行添加
func AddInitIocFunc(initIocFunc InitIocFunc) {
	initIocFuncs = append(initIocFuncs, initIocFunc)
}

// 添加初始化函数，由各个默认自行添加
func AddInitFunc(initFunc InitFunc) {
	initFuncs = append(initFuncs, initFunc)
}

// 系统启动时，调用各个模块的初始化函数
func InitOther() {
	// 调用各个默认ioc组件注册初始化，优先调用ioc初始化注册函数和注入函数（可能在后续的InitFunc中需要用到依赖实例）
	for _, initIocFunc := range initIocFuncs {
		initIocFunc()
	}
	initIocFuncs = nil

	// 为所有注册的实例注入其依赖的其他组件实例
	biz.ErrIsNil(ioc.InjectComponents())

	// 调用各个默认的初始化函数
	for _, initFunc := range initFuncs {
		go initFunc()
	}
	initFuncs = nil
}
