package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/ws"

	"github.com/gorilla/websocket"
)

type System struct {
}

func (s *System) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", s.ConnectWs).NoRes(),
	}
	return req.NewConfs("sysmsg", reqs[:]...)
}

// 连接websocket
func (s *System) ConnectWs(rc *req.Ctx) {
	wsConn, err := ws.Upgrader.Upgrade(rc.GetWriter(), rc.GetRequest(), nil)
	defer func() {
		if err := recover(); err != nil {
			errInfo := anyx.ToString(err)
			logx.Errorf("websocket connect error: %s", errInfo)
			if wsConn != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte(errInfo))
				wsConn.Close()
			}
		}
	}()

	biz.ErrIsNil(err)
	clientId := rc.Query("clientId")
	biz.NotEmpty(clientId, "clientId cannot be empty")

	// 权限校验
	err = req.PermissionHandler(rc)
	biz.ErrIsNil(err, "sys-websocket connect without permission")

	// 登录账号信息
	la := rc.GetLoginAccount()
	ws.AddClient(ws.UserId(la.Id), clientId, wsConn)
}
