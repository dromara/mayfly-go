package router

import "github.com/gin-gonic/gin"

func Init(router *gin.RouterGroup) {
	InitTagTreeRouter(router)
	InitTeamRouter(router)
	InitResourceAuthCertRouter(router)
	InitResourceOpLogRouter(router)
}
