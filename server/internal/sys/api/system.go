package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
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
			errInfo := anyx.ToString(err)
			logx.Errorf("websocket连接失败: %s", errInfo)
			if wsConn != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte(errInfo))
				wsConn.Close()
			}
		}
	}()

	biz.ErrIsNilAppendErr(err, "%s")
	clientId := g.Query("clientId")
	biz.NotEmpty(clientId, "clientId不能为空")

	// 权限校验
	rc := req.NewCtxWithGin(g)
	err = req.PermissionHandler(rc)
	biz.ErrIsNil(err, "sys websocket没有权限连接")

	// 登录账号信息
	la := rc.GetLoginAccount()
	ws.AddClient(ws.UserId(la.Id), clientId, wsConn)
}
