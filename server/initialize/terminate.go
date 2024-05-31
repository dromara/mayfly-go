package initialize

// 系统进程退出终止函数
type TerminateFunc func()

var (
	terminateFuncs = make([]TerminateFunc, 0)
)

// 添加系统退出终止时执行的函数，由各个默认自行添加
func AddTerminateFunc(terminateFunc TerminateFunc) {
	terminateFuncs = append(terminateFuncs, terminateFunc)
}

// 终止进程服务后的一些操作
func Terminate() {
	for _, terminateFunc := range terminateFuncs {
		terminateFunc()
	}
}
