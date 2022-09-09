package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitDbSqlExecRouter(router *gin.RouterGroup) {
	db := router.Group("/dbs/:dbId/sql-execs")
	{
		d := &api.DbSqlExec{
			DbSqlExecApp: application.GetDbSqlExecApp(),
		}
		// 获取所有数据库sql执行记录列表
		db.GET("", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c)
			rc.Handle(d.DbSqlExecs)
		})
	}
}
