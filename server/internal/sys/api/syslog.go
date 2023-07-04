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
	g := rc.GinCtx
	condition := &entity.Syslog{
		Type:        int8(ginx.QueryInt(g, "type", 0)),
		CreatorId:   uint64(ginx.QueryInt(g, "creatorId", 0)),
		Description: ginx.Query(g, "description", ""),
	}
	rc.ResData = r.SyslogApp.GetPageList(condition, ginx.GetPageParam(g), new([]entity.Syslog), "create_time DESC")
}
