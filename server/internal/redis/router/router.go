package router

import "github.com/gin-gonic/gin"

func Init(router *gin.RouterGroup) {
	InitRedisRouter(router)
}
