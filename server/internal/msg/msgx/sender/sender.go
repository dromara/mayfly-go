package sender

import "mayfly-go/internal/msg/msgx"

func init() {
	msgx.RegisterMsgSender(msgx.ChannelTypeEmail, EmailSender{})
	msgx.RegisterMsgSender(msgx.ChannelTypeDingBot, DingBotSender{})
	msgx.RegisterMsgSender(msgx.ChannelTypeQywxBot, QywxBotSender{})
	msgx.RegisterMsgSender(msgx.ChannelTypeFeishuBot, FeishuBotSender{})
}
