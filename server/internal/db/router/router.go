package router

import "github.com/gin-gonic/gin"

func Init(router *gin.RouterGroup) {
	InitInstanceRouter(router)
	InitDbRouter(router)
	InitDbSqlRouter(router)
	InitDbSqlExecRouter(router)
	InitDbBackupRouter(router)
	InitDbRestoreRouter(router)
	InitDbDataSyncRouter(router)
}
