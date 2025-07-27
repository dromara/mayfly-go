package sender

import (
	"context"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/ws"

	"github.com/spf13/cast"
)

type WsSender struct{}

func (e WsSender) Send(ctx context.Context, channel *msgx.Channel, msg *msgx.Msg) error {
	var err error
	content := msg.Content

	// 存在i18n msgId，content则使用msgId翻译
	if msgId := msg.TmplExtra.GetInt("msgId"); msgId != 0 {
		content = i18n.TC(ctx, i18n.MsgId(msgId))
	}
	if content != "" {
		content, err = stringx.TemplateParse(content, msg.Params)
		if err != nil {
			return err
		}
	}

	jsonMsg := msg.TmplExtra
	jsonMsg["msg"] = content
	jsonMsg["title"] = msg.Title
	jsonMsg["params"] = msg.Params

	for _, receiver := range msg.Receivers {
		ws.SendJsonMsg(ws.UserId(receiver.Id), cast.ToString(msg.Params["clientId"]), jsonMsg)
	}

	return nil
}
