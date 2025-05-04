package sender

import (
	"errors"
	"fmt"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/httpx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
)

type qywxBotMsgReq struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content             string   `json:"content"`
		MentionedMobileList []string `json:"mentioned_mobile_list"`
	} `json:"text"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

type qywxBotMsgResp struct {
	Code    int    `json:"errcode"`
	Message string `json:"errmsg"`
}

// QywxBotSender 企业微信机器人消息发送
type QywxBotSender struct{}

func (e QywxBotSender) Send(channel *msgx.Channel, msg *msgx.Msg) error {
	// https://developer.work.weixin.qq.com/document/path/91770
	msgReq := qywxBotMsgReq{}

	params := msg.Params
	receiver := ""
	// 使用receiver参数替换消息内容中可能存在的接收人信息
	if len(msg.Receivers) > 0 {
		if to := collx.ArrayMapFilter(msg.Receivers, func(a msgx.Receiver) (string, bool) {
			if uid := a.GetExtraString("qywxUserId"); uid != "" {
				// 使用<@userId>用于@指定用户
				return fmt.Sprintf("<@%s>", uid), true
			}
			return "", false
		}); len(to) > 0 {
			receiver = strings.Join(to, "")
		}
	}
	params[msgx.ReceiverKey] = receiver
	content, err := stringx.TemplateResolve(msg.Content, params)
	if err != nil {
		return err
	}

	if msg.Type == msgx.MsgTypeMarkdown {
		msgReq.MsgType = "markdown"
		msgReq.Markdown.Content = content
		// msgReq.Markdown.MentionedMobileList = receivers // markdown不支持@人,需要使用<@userId>
	} else {
		msgReq.MsgType = "text"
		msgReq.Text.Content = content

		// receivers := msg.Receivers
		// if len(msg.Receivers) == 0 {
		// 	receivers = []string{"@all"}
		// }

		// msgReq.Text.MentionedMobileList = receivers
	}

	var res qywxBotMsgResp
	err = httpx.NewReq(channel.URL).PostObj(msgReq).BodyTo(&res)
	if err != nil {
		return err
	}

	if res.Code != 0 {
		return errors.New(res.Message)
	}
	return nil
}
