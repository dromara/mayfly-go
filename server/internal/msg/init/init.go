package init

import (
	"context"
	"mayfly-go/initialize"
	"mayfly-go/internal/msg/api"
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/infra/persistence"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/internal/pkg/event"
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
	// 注册站内消息发送器
	msgx.RegisterMsgSender(msgx.ChannelTypeSiteMsg, application.GetMsgApp())

	msgTmplBizApp := ioc.Get[application.MsgTmplBiz]("MsgTmplBizApp")

	global.EventBus.SubscribeAsync(event.EventTopicBizMsgTmplSend, "BizMsgTmplSend", func(ctx context.Context, event *eventbus.Event[any]) error {
		return msgTmplBizApp.Send(ctx, event.Val.(dto.BizMsgTmplSend))
	}, false)

	msgTmplApp := ioc.Get[application.MsgTmpl]("MsgTmplApp")

	global.EventBus.SubscribeAsync(event.EventTopicMsgTmplSend, "MsgTmplSend", func(ctx context.Context, event *eventbus.Event[any]) error {
		eventVal := event.Val.(*dto.MsgTmplSendEvent)
		return msgTmplApp.SendMsg(ctx, &dto.MsgTmplSend{
			Tmpl:        eventVal.TmplChannel.Tmpl,
			Channels:    eventVal.TmplChannel.Channels,
			Params:      eventVal.Params,
			ReceiverIds: eventVal.ReceiverIds,
		})
	}, false)
}
