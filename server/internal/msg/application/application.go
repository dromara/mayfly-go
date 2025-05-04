package application

import (
	_ "mayfly-go/internal/msg/msgx/sender" // 注册消息发送器
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(msgAppImpl), ioc.WithComponentName("MsgApp"))
	ioc.Register(new(msgChannelAppImpl), ioc.WithComponentName("MsgChannelApp"))
	ioc.Register(new(msgTmplAppImpl), ioc.WithComponentName("MsgTmplApp"))
	ioc.Register(new(msgTmplBizAppImpl), ioc.WithComponentName("MsgTmplBizApp"))
}

func GetMsgApp() Msg {
	return ioc.Get[Msg]("MsgApp")
}
