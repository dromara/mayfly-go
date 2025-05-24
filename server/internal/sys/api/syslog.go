package api

import (
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type Syslog struct {
	syslogApp application.Syslog `inject:"T"`
}

func (s *Syslog) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", s.Syslogs),
		req.NewGet("/:id", s.SyslogDetail),
	}

	return req.NewConfs("syslogs", reqs[:]...)
}

func (r *Syslog) Syslogs(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.SysLogQuery](rc)
	res, err := r.syslogApp.GetPageList(queryCond, "create_time DESC")
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (r *Syslog) SyslogDetail(rc *req.Ctx) {
	rc.ResData = r.syslogApp.GetLogDetail(uint64(rc.PathParamInt("id")))
}
