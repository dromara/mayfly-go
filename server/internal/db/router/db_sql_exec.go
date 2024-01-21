package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitDbSqlExecRouter(router *gin.RouterGroup) {
	db := router.Group("/dbs/:dbId/sql-execs")

	d := new(api.DbSqlExec)
	biz.ErrIsNil(ioc.Inject(d))

	// 获取所有数据库sql执行记录列表
	req.NewGet("", d.DbSqlExecs).Group(db)

}
