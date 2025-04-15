package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Msg))
	ioc.Register(new(MsgChannel))
	ioc.Register(new(MsgTmpl))
}
