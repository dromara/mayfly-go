package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var Manager = NewClientManager() // 管理者

func init() {
	go Manager.Start()
}

// 添加ws客户端
func AddClient(userId UserId, clientId string, conn *websocket.Conn) *Client {
	if len(clientId) == 0 {
		return nil
	}
	cli := NewClient(UserId(userId), clientId, conn)
	cli.Read()
	Manager.AddClient(cli)
	return cli
}

func CloseClient(uid UserId) {
	Manager.CloseByUid(uid)
}

// 对指定用户发送json类型消息
func SendJsonMsg(userId UserId, clientId string, msg any) {
	Manager.SendJsonMsg(userId, clientId, msg)
}
