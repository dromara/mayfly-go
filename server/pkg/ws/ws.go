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
func AddClient(userId uint64, clientUuid string, conn *websocket.Conn) *Client {
	if len(clientUuid) == 0 {
		return nil
	}
	cli := NewClient(UserId(userId), clientUuid, conn)
	cli.Read()
	Manager.AddClient(cli)
	return cli
}

func CloseClient(clientUuid string) {
	Manager.CloseByClientUuid(clientUuid)
}

// 对指定用户发送json类型消息
func SendJsonMsg(clientUuid string, msg any) {
	Manager.SendJsonMsg(clientUuid, msg)
}
