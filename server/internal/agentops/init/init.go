package init

import (
	"context"

	"mayfly-go/internal/agentops/api"
	"mayfly-go/internal/agentops/application"
	"mayfly-go/internal/agentops/infra/persistence"
	"mayfly-go/internal/agentops/trigger"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/starter"
)

func init() {
	// 注册 agentops 模块的 IoC 组件（仓储 + 应用服务 + API）
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	// 启动 Run 有界工作池（在 DB 迁移建表之后执行；worker 认领前会等待 ai 运行时就绪）
	starter.AddInitFunc(application.Init)

	// 退出时优雅停止提案执行异步体与 Run 工作池：不接线则 SIGTERM 会在库里留下
	// 永久 running 的残留行，进而把同一任务的后续派发永久跳过（见 application.Terminate）
	starter.AddTerminateFunc(application.Terminate)

	// 装配事件驱动触发：注册内置触发源 + 订阅中立运维触发事件（告警联动入口）
	starter.AddInitFunc(initTriggerWiring)
}

// initTriggerWiring 注册内置触发源到默认注册表，并订阅全局事件总线的中立运维触发事件。
//
// 依赖不变式：agentops 仅经 internal/pkg/event 中立契约 + global.EventBus 消费 alert 发布的
// 事件，二者无直接依赖（无环）。订阅为异步：告警风暴下事件排队消费，不阻塞发布方主链路。
func initTriggerWiring() {
	trigger.InstallAlertSource(trigger.Default)
	trigger.InstallWebhookSource(trigger.Default)

	err := global.EventBus.SubscribeAsync(event.EventTopicOpsTrigger, "agentops-ops-trigger",
		func(ctx context.Context, e *eventbus.Event[any]) error {
			evt, ok := e.Val.(*event.OpsTriggerEvent)
			if !ok || evt == nil {
				// 非本契约事件（同主题异类型）直接忽略，不报错影响其他订阅者
				return nil
			}
			return application.GetTriggerService().Dispatch(ctx, evt)
		}, false)
	if err != nil {
		logx.ErrorTrace("subscribe ops trigger event failed", err)
	}
}
