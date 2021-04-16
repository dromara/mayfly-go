package middleware

import (
	"mayfly-go/base/ctx"

	"github.com/gin-gonic/gin"
)

var logHandler = new(ctx.LogInfo)

func ReqLog() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 处理请求
		c.Next()

		reqCtxI, exist := c.Get("reqCtx")
		if !exist {
			return
		}

		reqCtx := reqCtxI.(*ctx.ReqCtx)
		logHandler.AfterHandle(reqCtx)
	}
}
