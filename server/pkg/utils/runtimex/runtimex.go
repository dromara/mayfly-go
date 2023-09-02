package runtimex

import (
	"fmt"
	"runtime"
	"strings"
)

// 获取指定堆栈描述信息
//
// @param skip: 跳过堆栈个数
// @param nFrames: 需要描述的堆栈个数
func StatckStr(skip, nFrames int) string {
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
		fmt.Fprintf(&b, "called from %s (%s:%d)\n\t", frame.Function, ParseFrameFile(frame.File), frame.Line)
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

// 处理栈帧文件名
func ParseFrameFile(frameFile string) string {
	// 尝试将完整路径如/usr/local/.../mayfly-go/server/pkg/starter/web-server.go切割为pkg/starter/web-server.go
	if ss := strings.Split(frameFile, "mayfly-go/server/"); len(ss) > 1 {
		return ss[1]
	}
	return frameFile
}
