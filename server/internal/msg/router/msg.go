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
	{
		msg.GET("/self", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.GetMsgs)
		})
	}
}
