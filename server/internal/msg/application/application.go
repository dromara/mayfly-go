package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(msgAppImpl), ioc.WithComponentName("MsgApp"))
}

func GetMsgApp() Msg {
	return ioc.Get[Msg]("MsgApp")
}
