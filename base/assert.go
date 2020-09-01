package base

import (
	"reflect"
)

func BizErrIsNil(err error, msg string) {
	if err != nil {
		panic(NewBizErr(msg))
	}
}

func ErrIsNil(err error, msg string) {
	if err != nil {
		panic(err)
	}
}

func IsTrue(exp bool, msg string) {
	if !exp {
		panic(NewBizErr(msg))
	}
}

func NotEmpty(str string, msg string) {
	if str == "" {
		panic(NewBizErr(msg))
	}
}

func NotNil(data interface{}, msg string) {
	if reflect.ValueOf(data).IsNil() {
		panic(NewBizErr(msg))
	}
}

func Nil(data interface{}, msg string) {
	if !reflect.ValueOf(data).IsNil() {
		panic(NewBizErr(msg))
	}
}
