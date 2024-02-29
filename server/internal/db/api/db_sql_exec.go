package api

import (
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/conv"
	"strings"
)

type DbSqlExec struct {
	DbSqlExecApp application.DbSqlExec `inject:""`
}

func (d *DbSqlExec) DbSqlExecs(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage(rc, new(entity.DbSqlExecQuery))

	if statusStr := rc.Query("status"); statusStr != "" {
		queryCond.Status = collx.ArrayMap[string, int8](strings.Split(statusStr, ","), func(val string) int8 {
			return int8(conv.Str2Int(val, 0))
		})
	}
	res, err := d.DbSqlExecApp.GetPageList(queryCond, page, new([]entity.DbSqlExec))
	biz.ErrIsNil(err)
	rc.ResData = res
}
