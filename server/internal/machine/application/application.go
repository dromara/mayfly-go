package application

import (
	"context"
	"mayfly-go/internal/event"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/ioc"
	"sync"
)

func InitIoc() {
	ioc.Register(new(machineAppImpl), ioc.WithComponentName("MachineApp"))
	ioc.Register(new(machineFileAppImpl), ioc.WithComponentName("MachineFileApp"))
	ioc.Register(new(machineScriptAppImpl), ioc.WithComponentName("MachineScriptApp"))
	ioc.Register(new(machineCronJobAppImpl), ioc.WithComponentName("MachineCronJobApp"))
	ioc.Register(new(machineTermOpAppImpl), ioc.WithComponentName("MachineTermOpApp"))
	ioc.Register(new(machineCmdConfAppImpl), ioc.WithComponentName("MachineCmdConfApp"))
}

func Init() {
	sync.OnceFunc(func() {
		GetMachineCronJobApp().InitCronJob()

		GetMachineApp().TimerUpdateStats()

		GetMachineTermOpApp().TimerDeleteTermOp()

		global.EventBus.Subscribe(event.EventTopicDeleteMachine, "machineFile", func(ctx context.Context, event *eventbus.Event) error {
			me := event.Val.(*entity.Machine)
			return GetMachineFileApp().DeleteByCond(ctx, &entity.MachineFile{MachineId: me.Id})
		})

		global.EventBus.Subscribe(event.EventTopicDeleteMachine, "machineScript", func(ctx context.Context, event *eventbus.Event) error {
			me := event.Val.(*entity.Machine)
			return GetMachineScriptApp().DeleteByCond(ctx, &entity.MachineScript{MachineId: me.Id})
		})
	})()
}

func GetMachineApp() Machine {
	return ioc.Get[Machine]("MachineApp")
}

func GetMachineFileApp() MachineFile {
	return ioc.Get[MachineFile]("MachineFileApp")
}

func GetMachineScriptApp() MachineScript {
	return ioc.Get[MachineScript]("MachineScriptApp")
}

func GetMachineCronJobApp() MachineCronJob {
	return ioc.Get[MachineCronJob]("MachineCronJobApp")
}

func GetMachineTermOpApp() MachineTermOp {
	return ioc.Get[MachineTermOp]("MachineTermOpApp")
}
