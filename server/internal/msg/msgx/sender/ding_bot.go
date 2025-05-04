package sender

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/httpx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"net/url"
	"time"
)

type dingBotMsgReq struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	Markdown struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"markdown"`
	At struct {
		// AtUserIds []string `json:"atUserIds"`
		AtMobiles []string `json:"atMobiles"`
		IsAtAll   bool     `json:"isAtAll"`
	} `json:"at"`
}

type dingBotMsgResp struct {
	Code    int    `json:"errcode"`
	Message string `json:"errmsg"`
}

// DingBotSender 钉钉机器人消息发送
type DingBotSender struct{}

func (d DingBotSender) Send(channel *msgx.Channel, msg *msgx.Msg) error {
	// https://open.dingtalk.com/document/robots/custom-robot-access#title-72m-8ag-pqw
	msgReq := dingBotMsgReq{}

	params := msg.Params
	receiver := collx.ArrayMapFilter(msg.Receivers, func(a msgx.Receiver) (string, bool) {
		return a.Mobile, a.Mobile != ""
	})

	if len(receiver) > 0 {
		msgReq.At.AtMobiles = receiver
		// 替换文本中的receiver，使用@mobile用于@指定用户
		params[msgx.ReceiverKey] = strings.Join(collx.ArrayMap(receiver, func(a string) string { return "@" + a }), "")
	} else {
		msgReq.At.IsAtAll = true
		params[msgx.ReceiverKey] = ""
	}

	content, err := stringx.TemplateResolve(msg.Content, params)
	if err != nil {
		return err
	}

	if msg.Type == msgx.MsgTypeMarkdown {
		msgReq.MsgType = "markdown"
		msgReq.Markdown.Title = msg.Title
		msgReq.Markdown.Text = content
	} else {
		msgReq.MsgType = "text"
		msgReq.Text.Content = content
	}

	timestamp := time.Now().UnixMilli()
	sign, err := d.sign(channel.GetExtraString("secret"), timestamp)
	if err != nil {
		return err
	}

	var res dingBotMsgResp
	err = httpx.NewReq(fmt.Sprintf("%s&timestamp=%d&sign=%s", channel.URL, timestamp, sign)).
		PostObj(msgReq).
		BodyTo(&res)
	if err != nil {
		return err
	}

	if res.Code != 0 {
		return errors.New(res.Message)
	}
	return nil
}

func (d DingBotSender) sign(secret string, timestamp int64) (string, error) {
	// https://open.dingtalk.com/document/robots/customize-robot-security-settings
	// timestamp + key -> sha256 -> URL encode
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	_, err := h.Write([]byte(stringToSign))
	if err != nil {
		return "", err
	}
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	signature = url.QueryEscape(signature)
	return signature, nil
}
