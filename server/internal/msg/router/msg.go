package router

import (
	"mayfly-go/internal/msg/api"
	"mayfly-go/internal/msg/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMsgRouter(router *gin.RouterGroup) {
	msg := router.Group("msgs")
	a := &api.Msg{
		MsgApp: application.GetMsgApp(),
	}

	req.NewGet("/self", a.GetMsgs).Group(msg)
}
