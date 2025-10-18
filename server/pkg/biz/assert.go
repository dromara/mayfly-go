package biz

import (
	"context"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/utils/anyx"

	"reflect"
)

// 断言错误为ni
// @param msgAndParams 消息与参数占位符，第一位为错误消息可包含%s等格式化标识。其余为Sprintf格式化值内容
//
//	ErrIsNil(err)
//	ErrIsNil(err, "xxxx")
//	ErrIsNil(err, "xxxx: %s", "yyyy")
func ErrIsNil(err error, msgAndParams ...any) {
	if err != nil {
		if len(msgAndParams) == 0 {
			panic(errorx.NewBiz(err.Error()))
		}

		panic(errorx.NewBizf(msgAndParams[0].(string), msgAndParams[1:]...))
	}
}

// 断言错误为ni
// @param msgId i18n消息id
//
//	ErrIsNil(err)
//	ErrIsNil(err, "xxxx")
//	ErrIsNil(err, "xxxx: %s", "yyyy")
func ErrIsNilI(ctx context.Context, err error, msgId i18n.MsgId, attrs ...any) {
	if err != nil {
		if len(attrs) == 0 {
			panic(errorx.NewBiz(err.Error()))
		}

		panic(errorx.NewBiz(i18n.TC(ctx, msgId, attrs...)))
	}
}

func ErrNotNil(err error, msg string, params ...any) {
	if err == nil {
		panic(errorx.NewBizf(msg, params...))
	}
}

// ErrIsNilAppendErr 断言错误为nil，否则抛出业务异常，异常消息可包含‘%s’进行错误信息提示
//
//	// -> xxxx: err.Error()
//	biz.ErrIsNilAppendErr(err, "xxxx: %s")
func ErrIsNilAppendErr(err error, msg string) {
	if err != nil {
		panic(errorx.NewBizf(msg, err.Error()))
	}
}

// ErrIsNilAppendErr 断言错误为nil，否则抛出业务异常，异常消息可包含‘%s’进行错误信息提示
//
//	// -> xxxx: err.Error()
//	biz.ErrIsNilAppendErr(err, "xxxx: %s")
func ErrIsNilAppendErrI(ctx context.Context, err error, msgId i18n.MsgId) {
	if err != nil {
		panic(errorx.NewBizf(i18n.TC(ctx, msgId), err.Error()))
	}
}

func IsTrue(exp bool, msg string, params ...any) {
	if !exp {
		panic(errorx.NewBizf(msg, params...))
	}
}

func IsTrueI(ctx context.Context, exp bool, msgId i18n.MsgId, attrs ...any) {
	if !exp {
		panic(errorx.NewBiz(i18n.TC(ctx, msgId, attrs...)))
	}
}

func IsTrueBy(exp bool, err *errorx.BizError) {
	if !exp {
		panic(err)
	}
}

func NotEmpty(str string, msg string, params ...any) {
	if str == "" {
		panic(errorx.NewBizf(msg, params...))
	}
}

func NotNil(data any, msg string, params ...any) {
	if reflect.ValueOf(data).IsNil() {
		panic(errorx.NewBizf(msg, params...))
	}
}

func NotBlank(data any, msg string, params ...any) {
	if anyx.IsBlank(data) {
		panic(errorx.NewBizf(msg, params...))
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
