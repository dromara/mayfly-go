package api

import (
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"

	"github.com/spf13/cast"
)

type HisProcinstOp struct {
	hisProcinstOpApp application.HisProcinstOp `inject:"T"`
	// 时间线读接口的归属判定（与详情同一判据）
	procinstApp application.Procinst `inject:"T"`
}

func (p *HisProcinstOp) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet(":id", p.GetHisProcinstOps),
	}

	return req.NewConfs("/flow/his-procinsts-op", reqs[:]...)
}

func (p *HisProcinstOp) GetHisProcinstOps(rc *req.Ctx) {
	instId := cast.ToUint64(rc.PathParam("id"))
	// 时间线同样按实例归属收窄（与详情同一判据）
	biz.ErrIsNil(p.procinstApp.CanViewProcinst(rc.MetaCtx, instId))
	res, err := p.hisProcinstOpApp.ListByCond(&entity.HisProcinstOp{ProcinstId: instId})
	biz.ErrIsNil(err)
	rc.ResData = res
}
