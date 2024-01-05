package init

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/global"
)

func Init() {
	application.GetMachineCronJobApp().InitCronJob()

	application.GetMachineApp().TimerUpdateStats()

	global.EventBus.Subscribe(consts.DeleteMachineEventTopic, "machineFile", func(ctx context.Context, event *eventbus.Event) error {
		me := event.Val.(*entity.Machine)
		return application.GetMachineFileApp().DeleteByCond(ctx, &entity.MachineFile{MachineId: me.Id})
	})

	global.EventBus.Subscribe(consts.DeleteMachineEventTopic, "machineScript", func(ctx context.Context, event *eventbus.Event) error {
		me := event.Val.(*entity.Machine)
		return application.GetMachineScriptApp().DeleteByCond(ctx, &entity.MachineScript{MachineId: me.Id})
	})

	global.EventBus.Subscribe(consts.DeleteMachineEventTopic, "machineCronJob", func(ctx context.Context, event *eventbus.Event) error {
		me := event.Val.(*entity.Machine)
		var jobIds []uint64
		application.GetMachineCronJobApp().MachineRelateCronJobs(ctx, me.Id, jobIds)
		return nil
	})
}
