package gox

import (
	"errors"
	"mayfly-go/pkg/logx"
	"runtime/debug"
)

// RecoverPanic 捕获panic日志, 可选传入panic时的回调函数
func RecoverPanic(onPanic ...func(error)) {
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
