package init

import (
	"context"
	"mayfly-go/initialize"
	"mayfly-go/internal/event"
	"mayfly-go/internal/msg/api"
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/infrastructure/persistence"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/ioc"
)

func init() {
	initialize.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	initialize.AddInitFunc(Init)
}

func Init() {
	msgTmplBizApp := ioc.Get[application.MsgTmplBiz]("MsgTmplBizApp")

	global.EventBus.SubscribeAsync(event.EventTopicBizMsgTmplSend, "BizMsgTmplSend", func(ctx context.Context, event *eventbus.Event[any]) error {
		return msgTmplBizApp.Send(ctx, event.Val.(dto.BizMsgTmplSend))
	}, false)
}
