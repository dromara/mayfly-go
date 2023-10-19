package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type System struct {
}

// 连接websocket
func (s *System) ConnectWs(g *gin.Context) {
	wsConn, err := ws.Upgrader.Upgrade(g.Writer, g.Request, nil)
	defer func() {
		if err := recover(); err != nil {
			logx.ErrorTrace("websocket连接失败: ", err.(error))
			if wsConn != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte(err.(error).Error()))
				wsConn.Close()
			}
		}
	}()

	biz.ErrIsNilAppendErr(err, "%s")
	clientId := g.Query("clientId")
	biz.NotEmpty(clientId, "clientId不能为空")

	// 权限校验
	rc := req.NewCtxWithGin(g)
	if err = req.PermissionHandler(rc); err != nil {
		panic("sys ws连接没有权限")
	}

	// 登录账号信息
	la := rc.LoginAccount
	if la != nil {
		ws.AddClient(ws.UserId(la.Id), clientId, wsConn)
	}
}
