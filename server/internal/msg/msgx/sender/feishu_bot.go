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

	"time"

	"github.com/may-fly/cast"
)

type feishuBotMsgReq struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
	Timestamp string `json:"timestamp"`
	Sign      string `json:"sign"`
}

type feishuBotMsgResp struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// FeishuBotSender 发送飞书机器人消息
type FeishuBotSender struct{}

func (f FeishuBotSender) Send(channel *msgx.Channel, msg *msgx.Msg) error {
	// https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
	msgReq := feishuBotMsgReq{
		MsgType: "text",
	}

	params := msg.Params
	receiver := `<at user_id="all"></at>`
	// 使用receiver参数替换消息内容中可能存在的接收人信息
	if len(msg.Receivers) > 0 {
		if to := collx.ArrayMapFilter(msg.Receivers, func(a msgx.Receiver) (string, bool) {
			if uid := a.GetExtraString("feishuUserId"); uid != "" {
				// 使用<at user_id="userId"></at>
				return fmt.Sprintf(`<at user_id="%s"></at>`, uid), true
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

	msgReq.Content.Text = content

	if secret := channel.GetExtraString("secret"); secret != "" {
		timestamp := time.Now().Unix()
		if sign, err := f.sign(secret, timestamp); err != nil {
			return err
		} else {
			msgReq.Sign = sign
		}
		msgReq.Timestamp = cast.ToString(timestamp)
	}

	var res feishuBotMsgResp
	err = httpx.NewReq(channel.URL).
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

func (e FeishuBotSender) sign(secret string, timestamp int64) (string, error) {
	//timestamp + key 做sha256, 再进行base64 encode
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret
	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}
