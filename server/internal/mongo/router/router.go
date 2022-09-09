package router

import "github.com/gin-gonic/gin"

func Init(router *gin.RouterGroup) {
	InitMongoRouter(router)
}
