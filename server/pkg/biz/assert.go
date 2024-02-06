package biz

import (
	"fmt"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"

	"reflect"
)

// 断言错误为ni
// @param msgAndParams 消息与参数占位符，第一位为错误消息可包含%s等格式化标识。其余为Sprintf格式化值内容
//
// ErrIsNil(err)
// ErrIsNil(err, "xxxx")
// ErrIsNil(err, "xxxx: %s", "yyyy")
func ErrIsNil(err error, msgAndParams ...any) {
	if err != nil {
		if len(msgAndParams) == 0 {
			panic(errorx.NewBiz(err.Error()))
		}

		panic(errorx.NewBiz(fmt.Sprintf(msgAndParams[0].(string), msgAndParams[1:]...)))
	}
}

func ErrNotNil(err error, msg string, params ...any) {
	if err == nil {
		panic(errorx.NewBiz(fmt.Sprintf(msg, params...)))
	}
}

func ErrIsNilAppendErr(err error, msg string) {
	if err != nil {
		panic(errorx.NewBiz(fmt.Sprintf(msg, err.Error())))
	}
}

func IsTrue(exp bool, msg string, params ...any) {
	if !exp {
		panic(errorx.NewBiz(fmt.Sprintf(msg, params...)))
	}
}

func IsTrueBy(exp bool, err errorx.BizError) {
	if !exp {
		panic(err)
	}
}

func NotEmpty(str string, msg string, params ...any) {
	if str == "" {
		panic(errorx.NewBiz(fmt.Sprintf(msg, params...)))
	}
}

func NotNil(data any, msg string, params ...any) {
	if reflect.ValueOf(data).IsNil() {
		panic(errorx.NewBiz(fmt.Sprintf(msg, params...)))
	}
}

func NotBlank(data any, msg string, params ...any) {
	if anyx.IsBlank(data) {
		panic(errorx.NewBiz(fmt.Sprintf(msg, params...)))
	}
}

func IsEquals(data any, data1 any, msg string) {
	if data != data1 {
		panic(errorx.NewBiz(msg))
	}
}

func Nil(data any, msg string) {
	if !reflect.ValueOf(data).IsNil() {
		panic(errorx.NewBiz(msg))
	}
}
