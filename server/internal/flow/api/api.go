package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Procdef))
	ioc.Register(new(Procinst))
	ioc.Register(new(ProcinstTask))
	ioc.Register(new(HisProcinstOp))
}
