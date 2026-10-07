package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Machine))
	ioc.Register(new(MachineFile))
	ioc.Register(new(MachineScript))
	ioc.Register(new(MachineCronJob))
	ioc.Register(new(MachineCmdConf))
	ioc.Register(new(MachineHostKey))
	ioc.Register(new(MachineMetric))
	ioc.Register(new(MachineDisk))
	ioc.Register(new(MachineBatchFile))
}
