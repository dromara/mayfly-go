package gox

import (
	"context"
	"errors"
	"mayfly-go/pkg/logx"
	"runtime/debug"
)

// Go 启动安全协程，自动捕获并处理panic
// 该函数会在协程中执行传入的函数f，并使用recover捕获可能发生的panic
// 如果发生panic，会记录错误日志，同时执行传入的回调函数（如果有）
//
// 参数:
//   - f: 要在协程中执行的函数
//   - onPanic: 可选的panic回调函数，当发生panic时会被调用，可传入多个回调函数
func Go(f func(), onPanic ...func(error)) {
	go func() {
		defer Recover(onPanic...)
		f()
	}()
}

// GoCtx 启动安全协程（带上下文）
// 该函数会在协程中执行传入的函数fn，并使用recover捕获可能发生的panic
// 如果发生panic，会记录错误日志，同时执行传入的回调函数（如果有）
//
// 参数:
//   - ctx: 上下文对象，用于传递取消信号、超时等
//   - fn: 要在协程中执行的函数，接收上下文参数
//   - onPanic: 可选的panic回调函数，当发生panic时会被调用，可传入多个回调函数
func GoCtx(ctx context.Context, fn func(context.Context), onPanic ...func(error)) {
	go func() {
		defer Recover(onPanic...)
		fn(ctx)
	}()
}

// Recover 捕获panic日志, 可选传入panic时的回调函数
// 该函数应作为defer调用的一部分，用于捕获并处理panic
//
// 参数:
//   - onPanic: 可选的panic回调函数，当发生panic时会被调用，可传入多个回调函数
func Recover(onPanic ...func(error)) {
	if r := recover(); r != nil {
		logx.ErrorTrace("PANIC: ", r)

		if len(onPanic) == 0 {
			return
		}

		var err error
		switch x := r.(type) {
		case error:
			err = x
		case string:
			err = errors.New(x)
		default:
			err = errors.New("unknown panic value: " + string(debug.Stack()))
		}

		for _, fn := range onPanic {
			fn(err)
		}
	}
}
