package assert

import "fmt"

// 断言条件为真，不满足的panic
func IsTrue(condition bool, panicMsg string, params ...any) {
	if !condition {
		if len(params) != 0 {
			panic(fmt.Sprintf(panicMsg, params...))
		}
		panic(panicMsg)
	}
}

func State(condition bool, panicMsg string, params ...any) {
	IsTrue(condition, panicMsg, params...)
}

func NotEmpty(str string, panicMsg string, params ...any) {
	IsTrue(str != "", panicMsg, params...)
}
