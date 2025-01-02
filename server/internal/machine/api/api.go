package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Dashbord))
	ioc.Register(new(Machine))
	ioc.Register(new(MachineFile))
	ioc.Register(new(MachineScript))
	ioc.Register(new(MachineCronJob))
	ioc.Register(new(MachineCmdConf))
}
