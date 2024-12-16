package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Mongo))
	ioc.Register(new(Dashbord))
}
