package biz

import (
	"fmt"
	"mayfly-go/base/utils"

	"reflect"
)

func BizErrIsNil(err error, msg string, params ...interface{}) {
	if err != nil {
		panic(NewBizErr(fmt.Sprintf(msg, params...)))
	}
}

func ErrIsNil(err error, msg string) {
	if err != nil {
		panic(err)
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
