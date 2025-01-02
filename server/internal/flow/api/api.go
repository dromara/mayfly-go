package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Procdef))
	ioc.Register(new(Procinst))
}
