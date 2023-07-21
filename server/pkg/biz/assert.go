package biz

import (
	"fmt"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/utils/anyx"

	"reflect"
)

func ErrIsNil(err error, msg string, params ...any) {
	if err != nil {
		global.Log.Error(msg + ": " + err.Error())
		panic(NewBizErr(fmt.Sprintf(msg, params...)))
	}
}

func ErrIsNilAppendErr(err error, msg string) {
	if err != nil {
		panic(NewBizErr(fmt.Sprintf(msg, err.Error())))
	}
}

func IsNil(err error) {
	switch t := err.(type) {
	case *BizError:
		panic(t)
	case error:
		global.Log.Error("非业务异常: " + err.Error())
		panic(NewBizErr(fmt.Sprintf("非业务异常: %s", err.Error())))
	}
}

func IsTrue(exp bool, msg string, params ...any) {
	if !exp {
		panic(NewBizErr(fmt.Sprintf(msg, params...)))
	}
}

func IsTrueBy(exp bool, err BizError) {
	if !exp {
		panic(err)
	}
}

func NotEmpty(str string, msg string, params ...any) {
	if str == "" {
		panic(NewBizErr(fmt.Sprintf(msg, params...)))
	}
}

func NotNil(data any, msg string, params ...any) {
	if reflect.ValueOf(data).IsNil() {
		panic(NewBizErr(fmt.Sprintf(msg, params...)))
	}
}

func NotBlank(data any, msg string, params ...any) {
	if anyx.IsBlank(data) {
		panic(NewBizErr(fmt.Sprintf(msg, params...)))
	}
}

func IsEquals(data any, data1 any, msg string) {
	if data != data1 {
		panic(NewBizErr(msg))
	}
}

func Nil(data any, msg string) {
	if !reflect.ValueOf(data).IsNil() {
		panic(NewBizErr(msg))
	}
}
