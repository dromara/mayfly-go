package persistence

import (
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newMsgRepo(), ioc.WithComponentName("MsgRepo"))
}

func GetMsgRepo() repository.Msg {
	return ioc.Get[repository.Msg]("msgRepo")
}
