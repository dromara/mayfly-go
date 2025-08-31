package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Docker))
	ioc.Register(new(Container))
	ioc.Register(new(Image))
}
