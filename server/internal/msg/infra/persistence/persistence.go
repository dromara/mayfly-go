package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newMsgRepo())
	ioc.Register(newMsgChannelRepo())
	ioc.Register(newMsgTmplRepo())
	ioc.Register(newMsgTmplChannelRepo())
	ioc.Register(newMsgTmplBizRepo())
}
