package router

import (
	"mayfly-go/internal/msg/api"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMsgRouter(router *gin.RouterGroup) {
	msg := router.Group("msgs")

	a := new(api.Msg)
	ioc.Inject(a)

	req.NewGet("/self", a.GetMsgs).Group(msg)
}
