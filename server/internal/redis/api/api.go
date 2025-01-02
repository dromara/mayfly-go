package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Redis))
	ioc.Register(new(Dashbord))
}
