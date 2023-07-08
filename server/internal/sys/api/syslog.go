package api

import (
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
)

type Syslog struct {
	SyslogApp application.Syslog
}

func (r *Syslog) Syslogs(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.SysLogQuery](rc.GinCtx, new(entity.SysLogQuery))
	rc.ResData = r.SyslogApp.GetPageList(queryCond, page, new([]entity.SysLog), "create_time DESC")
}
