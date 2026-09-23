package api

import (
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"

	"strings"
)

type DbSQLExec struct {
	dbSQLExecApp application.DbSQLExec `inject:"T"`
}

func (d *DbSQLExec) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取所有数据库sql执行记录列表
		req.NewGet("/sql-execs", d.DbSQLExecs),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

func (d *DbSQLExec) DbSQLExecs(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.DbSQLExecQuery]()
	if statusStr := rc.Query("status"); statusStr != "" {
		queryCond.Status = collx.ArrayMap[string, int8](strings.Split(statusStr, ","), func(val string) int8 {
			return cast.ToInt8(val)
		})
	}
	res, err := d.dbSQLExecApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = res
}
