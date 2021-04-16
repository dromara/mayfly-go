package utils

import "runtime"

// 获取调用堆栈信息
func GetStackTrace() string {
	var buf [2 << 10]byte
	return string(buf[:runtime.Stack(buf[:], false)])
}
