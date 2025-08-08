package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newFileRepo(), ioc.WithComponentName("FileRepo"))
}
