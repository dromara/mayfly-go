package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newMsgRepo(), ioc.WithComponentName("MsgRepo"))
	ioc.Register(newMsgChannelRepo(), ioc.WithComponentName("MsgChannelRepo"))
	ioc.Register(newMsgTmplRepo(), ioc.WithComponentName("MsgTmplRepo"))
	ioc.Register(newMsgTmplChannelRepo(), ioc.WithComponentName("MsgTmplChannelRepo"))
	ioc.Register(newMsgTmplBizRepo(), ioc.WithComponentName("MsgTmplBizRepo"))
}
