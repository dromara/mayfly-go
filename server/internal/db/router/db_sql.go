package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitDbSqlRouter(router *gin.RouterGroup) {
	db := router.Group("dbs")

	dbSql := &api.DbSql{
		DbSqlApp: application.GetDbSqlApp(),
	}

	reqs := [...]*req.Conf{

		// 用户sql相关
		req.NewPost(":dbId/sql", dbSql.SaveSql),

		req.NewGet(":dbId/sql", dbSql.GetSql),

		req.NewDelete(":dbId/sql", dbSql.DeleteSql),

		req.NewGet(":dbId/sql-names", dbSql.GetSqlNames),
	}

	req.BatchSetGroup(db, reqs[:])

}
