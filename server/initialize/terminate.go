package initialize

import (
	dbInit "mayfly-go/internal/db/init"
)

// 终止进程服务后的一些操作
func Terminate() {
	dbInit.Terminate()
}
