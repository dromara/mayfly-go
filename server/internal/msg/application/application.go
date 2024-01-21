package application

import (
	"mayfly-go/internal/msg/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func init() {
	persistence.Init()

	ioc.Register(new(msgAppImpl), ioc.WithComponentName("MsgApp"))
}

func GetMsgApp() Msg {
	return ioc.Get[Msg]("MsgApp")
}
