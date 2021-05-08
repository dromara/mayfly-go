package biz

import (
	"fmt"
	"mayfly-go/base/global"
	"mayfly-go/base/utils"

	"reflect"
)

func ErrIsNil(err error, msg string, params ...interface{}) {
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

func IsTrue(exp bool, msg string, params ...interface{}) {
	if !exp {
		panic(NewBizErr(fmt.Sprintf(msg, params...)))
	}
}

func IsTrueBy(exp bool, err BizError) {
	if !exp {
		panic(err)
	}
}

func NotEmpty(str string, msg string, params ...interface{}) {
	if str == "" {
		panic(NewBizErr(fmt.Sprintf(msg, params...)))
	}
}

func NotNil(data interface{}, msg string) {
	if reflect.ValueOf(data).IsNil() {
		panic(NewBizErr(msg))
	}
}

func NotBlank(data interface{}, msg string) {
	if utils.IsBlank(reflect.ValueOf(data)) {
		panic(NewBizErr(msg))
	}
}

func IsEquals(data interface{}, data1 interface{}, msg string) {
	if data != data1 {
		panic(NewBizErr(msg))
	}
}

func Nil(data interface{}, msg string) {
	if !reflect.ValueOf(data).IsNil() {
		panic(NewBizErr(msg))
	}
}
