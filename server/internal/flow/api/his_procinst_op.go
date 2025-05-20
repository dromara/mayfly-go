package api

import (
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type HisProcinstOp struct {
	hisProcinstOpApp application.HisProcinstOp `inject:"T"`
	procinstTaskApp  application.ProcinstTask  `inject:"T"`
}

func (p *HisProcinstOp) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet(":id", p.GetHisProcinstOps),
	}

	return req.NewConfs("/flow/his-procinsts-op", reqs[:]...)
}

func (p *HisProcinstOp) GetHisProcinstOps(rc *req.Ctx) {
	res, err := p.hisProcinstOpApp.ListByCond(&entity.HisProcinstOp{ProcinstId: uint64(rc.PathParamInt("id"))})
	biz.ErrIsNil(err)
	rc.ResData = res
}
