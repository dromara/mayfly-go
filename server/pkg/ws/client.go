package ws

import (
	"encoding/json"
	"errors"
	"mayfly-go/pkg/logx"
	"time"

	"github.com/gorilla/websocket"
)

type UserId uint64

// 客户端读取消息处理函数
// @param msg
type ReadMsgHandlerFunc func([]byte)

type Client struct {
	ClientId string          // 标识ID
	UserId   UserId          // 用户ID
	WsConn   *websocket.Conn // 用户连接

	ReadMsgHandler ReadMsgHandlerFunc // 读取消息处理函数
}

func NewClient(userId UserId, clientId string, socket *websocket.Conn) *Client {
	cli := &Client{
		ClientId: clientId,
		UserId:   userId,
		WsConn:   socket,
	}

	return cli
}

func (c *Client) WithReadHandlerFunc(readMsgHandlerFunc ReadMsgHandlerFunc) *Client {
	c.ReadMsgHandler = readMsgHandlerFunc
	return c
}

// 读取ws客户端消息
func (c *Client) Read() {
	go func() {
		for {
			if c.WsConn == nil {
				return
			}
			messageType, data, err := c.WsConn.ReadMessage()
			if err != nil {
				if messageType == -1 && websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
					Manager.CloseClient(c)
					return
				}
				if messageType != websocket.PingMessage {
					return
				}
			}
			if c.ReadMsgHandler != nil {
				c.ReadMsgHandler(data)
			}
		}
	}()
}

// 向客户端写入消息
func (c *Client) WriteMsg(msg *Msg) error {
	logx.Debugf("发送消息: toUserId=%v, toClientId=%s, data=%v", c.UserId, c.ClientId, msg.Data)

	if msg.Type == JsonMsg {
		bytes, _ := json.Marshal(msg.Data)
		return c.WsConn.WriteMessage(websocket.TextMessage, bytes)
	}

	if msg.Type == BinaryMsg {
		if byteData, ok := msg.Data.([]byte); ok {
			return c.WsConn.WriteMessage(websocket.BinaryMessage, byteData)
		} else {
			return errors.New("该数据不为数组类型")
		}
	}

	if msg.Type == TextMsg {
		if strData, ok := msg.Data.(string); ok {
			return c.WsConn.WriteMessage(websocket.TextMessage, []byte(strData))
		} else {
			return errors.New("该数据类型不为字符串")
		}
	}
	return errors.New("不存在该消息类型, 无法发送")
}

// 向客户写入ping消息
func (c *Client) Ping() error {
	return c.WsConn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
}
