package api

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ws"

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
			wsConn.WriteMessage(websocket.TextMessage, []byte(err.(error).Error()))
			wsConn.Close()
		}
	}()

	if err != nil {
		panic(biz.NewBizErr("升级websocket失败"))
	}
	// 权限校验
	rc := ctx.NewReqCtxWithGin(g)
	if err = ctx.PermissionHandler(rc); err != nil {
		panic(biz.NewBizErr("没有权限"))
	}
	// 登录账号信息
	la := rc.LoginAccount
	ws.Put(la.Id, wsConn)
}
