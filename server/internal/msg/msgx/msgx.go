package msgx

import (
	"context"
	"fmt"
	"mayfly-go/pkg/utils/collx"
	"sync"
)

type MsgType int8
type ChannelType string

const (
	MsgTypeText     MsgType = 1
	MsgTypeMarkdown MsgType = 2
	MsgTypeHtml     MsgType = 3
)

const (
	ChannelTypeSiteMsg   ChannelType = "siteMsg" // 站内消息
	ChannelTypeWs        ChannelType = "ws"      // websocket
	ChannelTypeEmail     ChannelType = "email"
	ChannelTypeDingBot   ChannelType = "dingBot"
	ChannelTypeQywxBot   ChannelType = "qywxBot"
	ChannelTypeFeishuBot ChannelType = "feishuBot"
)

const (
	ReceiverKey = "receiver"
)

// Send 发送消息
func Send(ctx context.Context, channel *Channel, msg *Msg) error {
	sender, err := GetMsgSender(channel.Type)
	if err != nil {
		return err
	}
	return sender.Send(ctx, channel, msg)
}

type Receiver struct {
	Extra collx.M

	Id     uint64 // 接收人ID
	Mobile string
	Email  string
}

type Msg struct {
	Title     string         // 消息title
	Type      MsgType        // 消息类型
	Content   string         // 模板消息内容
	Params    map[string]any // 消息参数(替换消息中的占位符)
	TmplExtra collx.M        // 模板消息额外信息

	Receivers []Receiver // 消息接收人
}

// Channel 消息发送渠道信息
type Channel struct {
	Extra collx.M

	Type ChannelType // 渠道类型
	Name string
	URL  string
}

// MsgSender 定义消息发送接口
type MsgSender interface {
	// Send 发送消息
	Send(ctx context.Context, channel *Channel, msg *Msg) error
}

var messageSenders sync.Map

// RegisterMsgSender 注册消息发送器
func RegisterMsgSender(channel ChannelType, sender MsgSender) {
	messageSenders.Store(channel, sender)
}

// GetMsgSender 获取消息发送器
func GetMsgSender(channel ChannelType) (MsgSender, error) {
	sender, ok := messageSenders.Load(channel)
	if !ok {
		return nil, fmt.Errorf("unsupported message channel %s", channel)
	}
	return sender.(MsgSender), nil
}
