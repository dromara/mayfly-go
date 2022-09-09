package router

import "github.com/gin-gonic/gin"

func Init(router *gin.RouterGroup) {
	InitDbRouter(router)
	InitDbSqlExecRouter(router)
}
