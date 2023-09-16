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
func AddClient(userId uint64, conn *websocket.Conn) *Client {
	cli := NewClient(UserId(userId), conn)
	cli.Read()
	Manager.AddClient(cli)
	return cli
}

func CloseClient(userid uint64) {
	Manager.CloseByUid(UserId(userid))
}

// 对指定用户发送消息
func SendMsg(userId uint64, msg *SysMsg) {
	Manager.SendJsonMsg(UserId(userId), msg)
}
