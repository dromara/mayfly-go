package ws

import (
	"encoding/json"
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

var conns = make(map[uint64]*websocket.Conn, 100)

// 放置ws连接
func Put(userId uint64, conn *websocket.Conn) {
	conns[userId] = conn
}

// 对指定用户发送消息
func SendMsg(userId uint64, msg *Msg) {
	conn := conns[userId]
	if conn != nil {
		bytes, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.TextMessage, bytes)
	}
}
