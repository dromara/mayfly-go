package persistence

import (
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newMsgRepo(), ioc.WithComponentName("MsgRepo"))
}
