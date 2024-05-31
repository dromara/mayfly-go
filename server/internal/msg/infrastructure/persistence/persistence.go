package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newMsgRepo(), ioc.WithComponentName("MsgRepo"))
}
