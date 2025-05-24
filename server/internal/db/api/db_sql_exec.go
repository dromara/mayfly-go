package api

import (
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"github.com/may-fly/cast"

	"strings"
)

type DbSqlExec struct {
	dbSqlExecApp application.DbSqlExec `inject:"T"`
}

func (d *DbSqlExec) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取所有数据库sql执行记录列表
		req.NewGet("/sql-execs", d.DbSqlExecs),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

func (d *DbSqlExec) DbSqlExecs(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.DbSqlExecQuery](rc)
	if statusStr := rc.Query("status"); statusStr != "" {
		queryCond.Status = collx.ArrayMap[string, int8](strings.Split(statusStr, ","), func(val string) int8 {
			return cast.ToInt8(val)
		})
	}
	res, err := d.dbSqlExecApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = res
}
