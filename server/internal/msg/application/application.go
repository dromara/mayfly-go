package application

import (
	_ "mayfly-go/internal/msg/msgx/sender" // 注册消息发送器
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(msgAppImpl))
	ioc.Register(new(msgChannelAppImpl))
	ioc.Register(new(msgTmplAppImpl))
	ioc.Register(new(msgTmplBizAppImpl))
}

func GetMsgApp() Msg {
	return ioc.Get[Msg]()
}

func GetMsgTmplApp() MsgTmpl {
	return ioc.Get[MsgTmpl]()
}
