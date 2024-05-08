package init

import (
	"context"
	"mayfly-go/initialize"
	"mayfly-go/internal/event"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/infrastructure/persistence"
	"mayfly-go/internal/machine/router"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/global"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
	})
	initialize.AddInitRouterFunc(router.Init)
	initialize.AddInitFunc(Init)
}

func Init() {
	application.GetMachineCronJobApp().InitCronJob()

	application.GetMachineApp().TimerUpdateStats()

	application.GetMachineTermOpApp().TimerDeleteTermOp()

	global.EventBus.Subscribe(event.EventTopicDeleteMachine, "machineFile", func(ctx context.Context, event *eventbus.Event) error {
		me := event.Val.(*entity.Machine)
		return application.GetMachineFileApp().DeleteByCond(ctx, &entity.MachineFile{MachineId: me.Id})
	})

	global.EventBus.Subscribe(event.EventTopicDeleteMachine, "machineScript", func(ctx context.Context, event *eventbus.Event) error {
		me := event.Val.(*entity.Machine)
		return application.GetMachineScriptApp().DeleteByCond(ctx, &entity.MachineScript{MachineId: me.Id})
	})
}
