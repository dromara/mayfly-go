package runtimex

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// StackStr 获取指定堆栈描述信息
//
//  -  skip: 跳过堆栈个数
//  -  nFrames: 需要描述的堆栈个数
func StackStr(skip, nFrames int) string {
	pcs := make([]uintptr, nFrames+1)
	n := runtime.Callers(skip+1, pcs)
	if n == 0 {
		return "(no stack)"
	}
	frames := runtime.CallersFrames(pcs[:n])
	var b strings.Builder
	i := 0
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&b, "called from %s (%s:%d)\n\t", frame.Function, filepath.Base(frame.File), frame.Line)
		if !more {
			break
		}
		i++
		if i >= nFrames {
			fmt.Fprintf(&b, "(rest of stack elided...)\n")
			break
		}
	}
	return b.String()
}
